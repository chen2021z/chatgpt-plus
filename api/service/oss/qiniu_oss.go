package oss

import (
	"bytes"
	"chatplus/core/types"
	"chatplus/utils"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
	"path/filepath"
	"time"
)

type QinNiuOss struct {
	config   *types.QiNiuOssConfig
	token    string
	uploader *storage.FormUploader
	manager  *storage.BucketManager
	proxyURL string
	dir      string
}

func NewQiNiuOss(appConfig *types.AppConfig) QinNiuOss {
	config := &appConfig.OSS.QiNiu
	// build storage uploader
	zone, ok := storage.GetRegionByID(storage.RegionID(config.Zone))
	if !ok {
		zone = storage.ZoneHuanan
	}
	storeConfig := storage.Config{Zone: &zone}
	formUploader := storage.NewFormUploader(&storeConfig)
	// generate token
	mac := qbox.NewMac(config.AccessKey, config.AccessSecret)
	putPolicy := storage.PutPolicy{
		Scope: config.Bucket,
	}
	return QinNiuOss{
		config:   config,
		token:    putPolicy.UploadToken(mac),
		uploader: formUploader,
		manager:  storage.NewBucketManager(mac, &storeConfig),
		proxyURL: appConfig.ProxyURL,
		dir:      "chatgpt-plus",
	}
}

func (s QinNiuOss) PutFile(ctx *gin.Context, name string) (string, error) {
	// 解析表单
	file, err := ctx.FormFile(name)
	if err != nil {
		return "", err
	}
	// 打开上传文件
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	fileExt := filepath.Ext(file.Filename)
	key := fmt.Sprintf("%s/%d%s", s.dir, time.Now().UnixMicro(), fileExt)
	// 上传文件
	ret := storage.PutRet{}
	extra := storage.PutExtra{}
	err = s.uploader.Put(ctx, &ret, s.token, key, src, file.Size, &extra)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", s.config.Domain, ret.Key), nil
}

func (s QinNiuOss) PutImg(imageURL string) (string, error) {
	imageData, err := utils.DownloadImage(imageURL, s.proxyURL)
	if err != nil {
		return "", fmt.Errorf("error with download image: %v", err)
	}
	fileExt := filepath.Ext(filepath.Base(imageURL))
	key := fmt.Sprintf("%s/%d%s", s.dir, time.Now().UnixMicro(), fileExt)
	ret := storage.PutRet{}
	extra := storage.PutExtra{}
	// 上传文件字节数据
	err = s.uploader.Put(context.Background(), &ret, s.token, key, bytes.NewReader(imageData), int64(len(imageData)), &extra)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", s.config.Domain, ret.Key), nil
}

func (s QinNiuOss) Delete(fileURL string) error {
	objectName := filepath.Base(fileURL)
	key := fmt.Sprintf("%s/%s", s.dir, objectName)
	return s.manager.Delete(s.config.Bucket, key)
}

var _ Uploader = QinNiuOss{}
