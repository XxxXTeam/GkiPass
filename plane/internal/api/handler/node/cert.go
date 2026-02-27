package node

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/service"
	"gkipass/plane/internal/types"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NodeCertHandler 节点证书处理器
type NodeCertHandler struct {
	app *types.App
}

// NewNodeCertHandler 创建节点证书处理器
func NewNodeCertHandler(app *types.App) *NodeCertHandler {
	return &NodeCertHandler{app: app}
}

// GenerateCert 生成节点证书
func (h *NodeCertHandler) GenerateCert(c *gin.Context) {
	nodeID := c.Param("id")

	// 获取节点信息
	node, err := h.app.DAO.GetNode(nodeID)
	if err != nil || node == nil {
		response.GinNotFound(c, "Node not found")
		return
	}

	// 权限检查（仅管理员可管理证书）
	if !middleware.IsAdmin(c) {
		response.GinForbidden(c, "No permission")
		return
	}

	// 初始化证书管理器
	certManager, err := service.NewNodeCertManager(h.app.DAO, "./certs")
	if err != nil {
		logger.Error("初始化证书管理器失败", zap.Error(err))
		response.InternalError(c, "Failed to initialize cert manager")
		return
	}

	// 生成证书
	cert, err := certManager.GenerateNodeCert(node.ID, node.Name)
	if err != nil {
		logger.Error("生成节点证书失败", zap.Error(err))
		response.InternalError(c, "Failed to generate certificate")
		return
	}

	response.SuccessWithMessage(c, "Certificate generated successfully", gin.H{
		"cert_id":     cert.ID,
		"expires_at":  cert.NotAfter,
		"common_name": cert.CommonName,
	})
}

// DownloadCert 下载节点证书
func (h *NodeCertHandler) DownloadCert(c *gin.Context) {
	nodeID := c.Param("id")

	// 获取节点信息
	node, err := h.app.DAO.GetNode(nodeID)
	if err != nil || node == nil {
		response.GinNotFound(c, "Node not found")
		return
	}

	// 权限检查
	if !middleware.IsAdmin(c) {
		response.GinForbidden(c, "No permission")
		return
	}

	// 查找节点证书
	nodeCerts, _ := h.app.DAO.ListCertificates("", nil)
	var certID string
	for _, cert := range nodeCerts {
		if cert.NodeID == nodeID && !cert.Revoked {
			certID = cert.ID
			break
		}
	}
	if certID == "" {
		response.GinBadRequest(c, "Node certificate not generated")
		return
	}

	// 初始化证书管理器
	certManager, err := service.NewNodeCertManager(h.app.DAO, "./certs")
	if err != nil {
		response.InternalError(c, "Failed to initialize cert manager")
		return
	}

	// 获取证书路径
	certPath, keyPath, caPath := certManager.GetNodeCertPath(node.ID)

	// 检查文件是否存在
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		response.GinNotFound(c, "Certificate files not found")
		return
	}

	// 创建 ZIP 文件
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// 添加证书文件
	files := map[string]string{
		"cert.pem":    certPath,
		"key.pem":     keyPath,
		"ca-cert.pem": caPath,
	}

	for name, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Error("读取证书文件失败", zap.String("path", path), zap.Error(err))
			continue
		}

		f, err := zipWriter.Create(name)
		if err != nil {
			logger.Error("创建ZIP文件失败", zap.Error(err))
			continue
		}

		if _, err := f.Write(data); err != nil {
			logger.Error("写入ZIP文件失败", zap.Error(err))
			continue
		}
	}

	// 获取证书信息
	nodeCertInfo, _ := h.app.DAO.GetCertificate(certID)
	genTime := "Unknown"
	if nodeCertInfo != nil {
		genTime = nodeCertInfo.NotBefore.Format("2006-01-02 15:04:05")
	}

	// 添加 README
	readme := fmt.Sprintf(`GKI Pass 节点证书包

节点ID: %s
节点名称: %s
生成时间: %s

文件说明:
- cert.pem: 节点证书
- key.pem: 节点私钥
- ca-cert.pem: CA 根证书

使用方法:
1. 将证书文件放置到节点的 certs/ 目录
2. 启动节点: ./client --token <your-connection-key> --cert-dir ./certs

注意事项:
- 请妥善保管私钥文件（key.pem）
- 证书有效期: 1年
- 证书到期前请及时续期
`, node.ID, node.Name, genTime)

	f, _ := zipWriter.Create("README.txt")
	f.Write([]byte(readme))

	if err := zipWriter.Close(); err != nil {
		response.InternalError(c, "Failed to create ZIP file")
		return
	}

	// 发送 ZIP 文件
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=node-%s-certs.zip", node.ID))
	c.Data(200, "application/zip", buf.Bytes())

	logger.Info("节点证书已下载", zap.String("nodeID", node.ID))
}

// RenewCert 续期节点证书
func (h *NodeCertHandler) RenewCert(c *gin.Context) {
	nodeID := c.Param("id")

	node, err := h.app.DAO.GetNode(nodeID)
	if err != nil || node == nil {
		response.GinNotFound(c, "Node not found")
		return
	}

	if !middleware.IsAdmin(c) {
		response.GinForbidden(c, "No permission")
		return
	}

	/* 查找当前有效证书 ID */
	oldCertID := ""
	certs, _ := h.app.DAO.ListCertificatesByNode(nodeID)
	for _, c := range certs {
		if !c.Revoked {
			oldCertID = c.ID
			break
		}
	}

	certManager, err := service.NewNodeCertManager(h.app.DAO, "./certs")
	if err != nil {
		response.InternalError(c, "Failed to initialize cert manager")
		return
	}

	newCert, err := certManager.RenewNodeCert(node.ID, node.Name, oldCertID)
	if err != nil {
		logger.Error("续期节点证书失败", zap.Error(err))
		response.InternalError(c, "Failed to renew certificate")
		return
	}

	response.SuccessWithMessage(c, "Certificate renewed successfully", gin.H{
		"cert_id":     newCert.ID,
		"expires_at":  newCert.NotAfter,
		"common_name": newCert.CommonName,
	})
}

// GetCertInfo 获取节点证书信息
func (h *NodeCertHandler) GetCertInfo(c *gin.Context) {
	nodeID := c.Param("id")

	node, err := h.app.DAO.GetNode(nodeID)
	if err != nil || node == nil {
		response.GinNotFound(c, "Node not found")
		return
	}

	if !middleware.IsAdmin(c) {
		response.GinForbidden(c, "No permission")
		return
	}

	/* 查找节点的有效证书 */
	nodeCerts, _ := h.app.DAO.ListCertificatesByNode(nodeID)
	var activeCert *models.NodeCertificate
	for i := range nodeCerts {
		if !nodeCerts[i].Revoked {
			activeCert = &nodeCerts[i]
			break
		}
	}

	if activeCert == nil {
		response.GinSuccess(c, gin.H{
			"has_cert": false,
			"message":  "Certificate not generated",
		})
		return
	}

	certManager, err := service.NewNodeCertManager(h.app.DAO, "./certs")
	if err != nil {
		response.InternalError(c, "Failed to initialize cert manager")
		return
	}

	certPath, _, _ := certManager.GetNodeCertPath(node.ID)
	fileExists := false
	if _, err := os.Stat(certPath); err == nil {
		fileExists = true
	}

	response.GinSuccess(c, gin.H{
		"has_cert":     true,
		"file_exists":  fileExists,
		"cert_id":      activeCert.ID,
		"common_name":  activeCert.CommonName,
		"not_before":   activeCert.NotBefore,
		"not_after":    activeCert.NotAfter,
		"revoked":      activeCert.Revoked,
		"expires_soon": certManager.CheckCertExpiry(activeCert),
		"cert_path":    filepath.Join("./certs/nodes", node.ID),
	})
}
