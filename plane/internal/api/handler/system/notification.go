package system

import (
	"strconv"

	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/types"

	"github.com/gin-gonic/gin"
)

// NotificationHandler 通知处理器
type NotificationHandler struct {
	app *types.App
}

// NewNotificationHandler 创建通知处理器
func NewNotificationHandler(app *types.App) *NotificationHandler {
	return &NotificationHandler{app: app}
}

// List 获取通知列表
func (h *NotificationHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	/* 分页参数边界保护 */
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	notifications, total, err := h.app.DAO.ListNotifications(userID, page, limit)
	if err != nil {
		response.InternalError(c, "Failed to list notifications")
		return
	}

	response.GinSuccess(c, gin.H{
		"data":        notifications,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// MarkAsRead 标记为已读
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	id := c.Param("id")

	if err := h.app.DAO.MarkNotificationAsRead(id); err != nil {
		response.InternalError(c, "Failed to mark as read")
		return
	}

	response.SuccessWithMessage(c, "Marked as read", nil)
}

// MarkAllAsRead 全部标记为已读
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if err := h.app.DAO.MarkAllNotificationsAsRead(userID); err != nil {
		response.InternalError(c, "Failed to mark all as read")
		return
	}

	response.SuccessWithMessage(c, "All marked as read", nil)
}

// Delete 删除通知
func (h *NotificationHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.app.DAO.DeleteNotification(id); err != nil {
		response.InternalError(c, "Failed to delete notification")
		return
	}

	response.SuccessWithMessage(c, "Notification deleted", nil)
}

// CreateNotificationRequest 创建通知请求
type CreateNotificationRequest struct {
	UserID   string `json:"user_id"` // 空表示全局通知
	Type     string `json:"type" binding:"required"`
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Link     string `json:"link"`
	Priority string `json:"priority"`
}

// Create 创建通知（管理员）
func (h *NotificationHandler) Create(c *gin.Context) {
	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if req.Priority == "" {
		req.Priority = "normal"
	}

	notification := &models.Notification{
		UserID:  req.UserID,
		Type:    req.Type,
		Title:   req.Title,
		Content: req.Content,
		Level:   req.Priority,
		Read:    false,
	}

	if err := h.app.DAO.CreateNotification(notification); err != nil {
		response.InternalError(c, "Failed to create notification")
		return
	}

	response.SuccessWithMessage(c, "Notification created", notification)
}
