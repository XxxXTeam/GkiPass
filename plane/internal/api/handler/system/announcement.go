package system

import (
	"strconv"
	"time"

	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/api/middleware"
	"gkipass/plane/internal/api/response"
	"gkipass/plane/internal/types"

	"github.com/gin-gonic/gin"
)

// AnnouncementHandler 公告处理器
type AnnouncementHandler struct {
	app *types.App
}

// NewAnnouncementHandler 创建公告处理器
func NewAnnouncementHandler(app *types.App) *AnnouncementHandler {
	return &AnnouncementHandler{app: app}
}

// ListActiveAnnouncements 获取有效公告列表（用户）
func (h *AnnouncementHandler) ListActiveAnnouncements(c *gin.Context) {
	announcements, err := h.app.DAO.ListActiveAnnouncements()
	if err != nil {
		response.InternalError(c, "Failed to list announcements")
		return
	}

	response.GinSuccess(c, announcements)
}

// GetAnnouncement 获取公告详情
func (h *AnnouncementHandler) GetAnnouncement(c *gin.Context) {
	id := c.Param("id")

	announcement, err := h.app.DAO.GetAnnouncement(id)
	if err != nil {
		response.InternalError(c, "Failed to get announcement")
		return
	}

	if announcement == nil {
		response.GinNotFound(c, "Announcement not found")
		return
	}

	response.GinSuccess(c, announcement)
}

// ListAll 获取所有公告（管理员）
func (h *AnnouncementHandler) ListAll(c *gin.Context) {
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

	announcements, total, err := h.app.DAO.ListAnnouncements(page, limit)
	if err != nil {
		response.InternalError(c, "Failed to list announcements")
		return
	}

	response.GinSuccess(c, gin.H{
		"data":        announcements,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// CreateAnnouncementRequest 创建公告请求
type CreateAnnouncementRequest struct {
	Title     string `json:"title" binding:"required,min=1,max=200"`
	Content   string `json:"content" binding:"required,max=10000"`
	Type      string `json:"type" binding:"required,oneof=info warning maintenance update"`
	Priority  string `json:"priority" binding:"omitempty,oneof=low normal high urgent"`
	Enabled   bool   `json:"enabled"`
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
}

// Create 创建公告（管理员）
func (h *AnnouncementHandler) Create(c *gin.Context) {
	var req CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		response.GinBadRequest(c, "Invalid start_time format")
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		response.GinBadRequest(c, "Invalid end_time format")
		return
	}

	/* 结束时间必须晚于开始时间 */
	if !endTime.After(startTime) {
		response.GinBadRequest(c, "end_time 必须晚于 start_time")
		return
	}

	if req.Priority == "" {
		req.Priority = "normal"
	}

	announcement := &models.Announcement{
		Title:     req.Title,
		Content:   req.Content,
		Type:      req.Type,
		Priority:  0,
		Enabled:   req.Enabled,
		StartAt:   &startTime,
		EndAt:     &endTime,
		CreatedBy: userID,
	}

	if err := h.app.DAO.CreateAnnouncement(announcement); err != nil {
		response.InternalError(c, "Failed to create announcement")
		return
	}

	response.SuccessWithMessage(c, "Announcement created", announcement)
}

// Update 更新公告（管理员）
func (h *AnnouncementHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinBadRequest(c, "Invalid request: "+err.Error())
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		response.GinBadRequest(c, "Invalid start_time format")
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		response.GinBadRequest(c, "Invalid end_time format")
		return
	}

	/* 结束时间必须晚于开始时间 */
	if !endTime.After(startTime) {
		response.GinBadRequest(c, "end_time 必须晚于 start_time")
		return
	}

	existing, err := h.app.DAO.GetAnnouncement(id)
	if err != nil || existing == nil {
		response.GinNotFound(c, "Announcement not found")
		return
	}

	existing.Title = req.Title
	existing.Content = req.Content
	existing.Type = req.Type
	existing.Enabled = req.Enabled
	existing.StartAt = &startTime
	existing.EndAt = &endTime

	if err := h.app.DAO.UpdateAnnouncement(existing); err != nil {
		response.InternalError(c, "Failed to update announcement")
		return
	}

	response.SuccessWithMessage(c, "Announcement updated", existing)
}

// Delete 删除公告（管理员）
func (h *AnnouncementHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.app.DAO.DeleteAnnouncement(id); err != nil {
		response.InternalError(c, "Failed to delete announcement")
		return
	}

	response.SuccessWithMessage(c, "Announcement deleted", nil)
}
