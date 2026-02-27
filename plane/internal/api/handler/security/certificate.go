package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"time"

	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/types"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CertificateHandler 证书处理器
type CertificateHandler struct {
	app *types.App
}

// NewCertificateHandler 创建证书处理器
func NewCertificateHandler(app *types.App) *CertificateHandler {
	return &CertificateHandler{app: app}
}

// GenerateCARequest 生成CA证书请求
type GenerateCARequest struct {
	Name        string `json:"name" binding:"required,min=1,max=64"`
	CommonName  string `json:"common_name" binding:"required,min=1,max=128"`
	ValidYears  int    `json:"valid_years" binding:"omitempty,min=1,max=30"`
	Description string `json:"description" binding:"omitempty,max=512"`
}

// GenerateCA 生成CA证书
func (h *CertificateHandler) GenerateCA(c *gin.Context) {
	var req GenerateCARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if req.ValidYears == 0 {
		req.ValidYears = 10 // 默认10年
	}

	// 生成私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		response.InternalError(c, "Failed to generate private key")
		return
	}

	// 创建证书模板
	notBefore := time.Now()
	notAfter := notBefore.AddDate(req.ValidYears, 0, 0)

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   req.CommonName,
			Organization: []string{"GKIPass"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
	}

	// 自签名CA证书
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		response.InternalError(c, "Failed to create certificate")
		return
	}

	// 编码为PEM格式
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	// 计算SPKI Pin
	pin := calculateSPKIPin(&privateKey.PublicKey)

	// 保存到数据库
	cert := &models.NodeCertificate{
		Type:        "ca",
		CommonName:  req.CommonName,
		CertPEM:     string(certPEM),
		KeyPEM:      string(keyPEM),
		Fingerprint: pin,
		NotBefore:   notBefore,
		NotAfter:    notAfter,
	}

	if err := h.app.DAO.CreateCertificate(cert); err != nil {
		logger.Error("保存CA证书失败", zap.Error(err))
		response.InternalError(c, "Failed to save certificate")
		return
	}

	response.SuccessWithMessage(c, "CA certificate generated successfully", cert)
}

// GenerateLeafRequest 生成叶子证书请求
type GenerateLeafRequest struct {
	Name        string `json:"name" binding:"required"`
	CommonName  string `json:"common_name" binding:"required"`
	ParentID    string `json:"parent_id" binding:"required"`
	ValidDays   int    `json:"valid_days"`
	Description string `json:"description"`
}

// GenerateLeaf 生成叶子证书
func (h *CertificateHandler) GenerateLeaf(c *gin.Context) {
	var req GenerateLeafRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if req.ValidDays == 0 {
		req.ValidDays = 90 // 默认90天（短周期）
	}

	// 获取父证书(CA)
	parentCert, err := h.app.DAO.GetCertificate(req.ParentID)
	if err != nil || parentCert == nil {
		response.GinBadRequest(c, "Parent CA certificate not found")
		return
	}

	// 解析父证书
	parentBlock, _ := pem.Decode([]byte(parentCert.CertPEM))
	parentX509, _ := x509.ParseCertificate(parentBlock.Bytes)

	parentKeyBlock, _ := pem.Decode([]byte(parentCert.KeyPEM))
	parentKey, _ := x509.ParsePKCS1PrivateKey(parentKeyBlock.Bytes)

	// 生成新的私钥
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	// 创建证书模板
	notBefore := time.Now()
	notAfter := notBefore.AddDate(0, 0, req.ValidDays)

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   req.CommonName,
			Organization: []string{"GKIPass Node"},
		},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	// 用父证书签名
	derBytes, _ := x509.CreateCertificate(rand.Reader, &template, parentX509, &privateKey.PublicKey, parentKey)

	// 编码为PEM格式
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	// 计算SPKI Pin
	pin := calculateSPKIPin(&privateKey.PublicKey)

	// 保存到数据库
	cert := &models.NodeCertificate{
		Type:        "leaf",
		CommonName:  req.CommonName,
		CertPEM:     string(certPEM),
		KeyPEM:      string(keyPEM),
		Fingerprint: pin,
		NotBefore:   notBefore,
		NotAfter:    notAfter,
	}

	if err := h.app.DAO.CreateCertificate(cert); err != nil {
		logger.Error("保存叶子证书失败", zap.Error(err))
		response.InternalError(c, "Failed to save certificate")
		return
	}

	response.SuccessWithMessage(c, "Leaf certificate generated successfully", cert)
}

// List 列出证书
func (h *CertificateHandler) List(c *gin.Context) {
	certType := c.Query("type")
	revokedStr := c.Query("revoked")

	var revoked *bool
	if revokedStr != "" {
		val := revokedStr == "true"
		revoked = &val
	}

	certs, err := h.app.DAO.ListCertificates(certType, revoked)
	if err != nil {
		logger.Error("获取证书列表失败", zap.Error(err))
		response.InternalError(c, "Failed to list certificates")
		return
	}

	/* 私钥已通过 json:"-" 自动隐藏 */

	response.GinSuccess(c, gin.H{
		"certificates": certs,
		"total":        len(certs),
	})
}

// Get 获取证书详情
func (h *CertificateHandler) Get(c *gin.Context) {
	id := c.Param("id")
	showPrivate := c.Query("show_private") == "true"

	cert, err := h.app.DAO.GetCertificate(id)
	if err != nil {
		logger.Error("获取证书失败", zap.String("id", id), zap.Error(err))
		response.InternalError(c, "Failed to get certificate")
		return
	}

	if cert == nil {
		response.GinNotFound(c, "Certificate not found")
		return
	}

	/* showPrivate=true 时返回完整证书内容（含 PEM），否则隐藏 */
	if showPrivate {
		response.GinSuccess(c, gin.H{
			"id":          cert.ID,
			"node_id":     cert.NodeID,
			"type":        cert.Type,
			"common_name": cert.CommonName,
			"cert_pem":    cert.CertPEM,
			"key_pem":     cert.KeyPEM,
			"ca_pem":      cert.CAPem,
			"not_before":  cert.NotBefore,
			"not_after":   cert.NotAfter,
			"fingerprint": cert.Fingerprint,
			"revoked":     cert.Revoked,
			"created_at":  cert.CreatedAt,
			"updated_at":  cert.UpdatedAt,
		})
		return
	}

	response.GinSuccess(c, cert)
}

// Revoke 吊销证书
func (h *CertificateHandler) Revoke(c *gin.Context) {
	id := c.Param("id")

	if err := h.app.DAO.RevokeCertificate(id); err != nil {
		logger.Error("吁销证书失败", zap.String("id", id), zap.Error(err))
		response.InternalError(c, "Failed to revoke certificate")
		return
	}

	response.SuccessWithMessage(c, "Certificate revoked successfully", nil)
}

// Download 下载证书
func (h *CertificateHandler) Download(c *gin.Context) {
	id := c.Param("id")
	includeKey := c.Query("include_key") == "true"

	cert, err := h.app.DAO.GetCertificate(id)
	if err != nil || cert == nil {
		response.GinNotFound(c, "Certificate not found")
		return
	}

	filename := cert.CommonName
	if filename == "" {
		filename = cert.ID
	}

	if includeKey {
		content := "# Certificate\n" + cert.CertPEM + "\n# Private Key\n" + cert.KeyPEM
		c.Header("Content-Disposition", "attachment; filename="+filename+".pem")
		c.Data(200, "application/x-pem-file", []byte(content))
	} else {
		c.Header("Content-Disposition", "attachment; filename="+filename+".crt")
		c.Data(200, "application/x-pem-file", []byte(cert.CertPEM))
	}
}

// calculateSPKIPin 计算SPKI Pin (用于证书固定)
func calculateSPKIPin(pubKey *rsa.PublicKey) string {
	spki, _ := x509.MarshalPKIXPublicKey(pubKey)
	hash := sha256.Sum256(spki)
	return base64.StdEncoding.EncodeToString(hash[:])
}
