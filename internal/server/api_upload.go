package server

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"lark-robot/internal/larkbot"
)

type UploadAPI struct {
	larkClient *larkbot.LarkClient
}

func NewUploadAPI(larkClient *larkbot.LarkClient) *UploadAPI {
	return &UploadAPI{larkClient: larkClient}
}

// UploadImage handles image upload and returns the image_key from Lark.
func (api *UploadAPI) UploadImage(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	imageKey, err := api.larkClient.UploadImage(c.Request.Context(), file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"image_key": imageKey})
}

// fileTypeMap maps file extensions to Lark file types.
var fileTypeMap = map[string]string{
	".opus": "opus",
	".mp4":  "mp4",
	".pdf":  "pdf",
	".doc":  "doc",
	".docx": "doc",
	".xls":  "xls",
	".xlsx": "xls",
	".ppt":  "ppt",
	".pptx": "ppt",
}

// UploadFile handles file upload and returns the file_key from Lark.
func (api *UploadAPI) UploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	fileName := header.Filename
	ext := strings.ToLower(filepath.Ext(fileName))

	fileType := "stream"
	if ft, ok := fileTypeMap[ext]; ok {
		fileType = ft
	}

	// Allow overriding file_type via form field
	if ft := c.PostForm("file_type"); ft != "" {
		fileType = ft
	}

	fileKey, err := api.larkClient.UploadFile(c.Request.Context(), fileType, fileName, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file_key":  fileKey,
		"file_name": fileName,
		"file_type": fileType,
		"file_size": fmt.Sprintf("%d", header.Size),
	})
}
