package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/uploadcare/uploadcare-go/internal/codec"
	"github.com/uploadcare/uploadcare-go/internal/config"
	"github.com/uploadcare/uploadcare-go/ucare"
)

// FileParams holds all possible params for the file upload
type FileParams struct {
	authParams

	// Data (required) holds the data to be uploaded.
	//
	// It must be smaller than 100MB.
	// An attempt of reading a larger file raises a 413 error with the
	// respective description. If you want to upload larger files, please
	// use multipart upload API methods.
	Data io.ReadSeeker `form:"file"`
	// Name (required) holds uploaded file name
	Name string
	// ContentType is a data content type. It will be auto-detected if left
	// blank, if it can't be auto-detected it will fallback
	// to application/octet-stream
	ContentType string

	// ToStore sets the file storing behaviour
	ToStore *string `form:"UPLOADCARE_STORE"`
}

type authParams struct {
	PubKey    string  `form:"UPLOADCARE_PUB_KEY"`
	Signature *string `form:"signature"`
	ExpiresAt *int64  `form:"expire"`
}

// EncodeReq implementes ucare.ReqEncoder
func (d *FileParams) EncodeReq(req *http.Request) error {
	authFuncI := req.Context().Value(config.CtxAuthFuncKey)
	authFunc, ok := authFuncI.(ucare.UploadAPIAuthFunc)
	if !ok {
		return errors.New("auth func has a wrong signature")
	}
	d.PubKey, d.Signature, d.ExpiresAt = authFunc()

	formReader, contentType, err := codec.EncodeReqFormData(d)
	if err != nil {
		return fmt.Errorf("creating req form body: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Body = formReader

	return nil
}

// UploadFile uploads a file and return its unique id (uuid).
// Comply with the RFC7578 standard.
func (s service) UploadFile(
	ctx context.Context,
	params *FileParams,
) (string, error) {
	var resp struct{ File string }

	if err := s.svc.ResourceOp(
		ctx,
		http.MethodPost,
		directUploadPathFormat,
		params,
		&resp,
	); err != nil {
		return "", err
	}

	log.Debugf("uploaded file: %s", resp.File)

	return resp.File, nil
}
