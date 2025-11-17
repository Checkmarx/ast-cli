package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type UploadModel struct {
	URL string `json:"url"`
}

type UploadsHTTPWrapper struct {
	path string
}

type StartMultipartUploadResponse struct {
	ObjectName string `json:"objectName"`
	UploadID   string `json:"UploadID"`
}
type StartMultipartUploadRequest struct {
	FileSize int64 `json:"fileSize"`
}
type MultipartPresignedURL struct {
	ObjectName string `json:"objectName"`
	UploadID   string `json:"UploadID"`
	PartNumber int    `json:"partNumber"`
}

type CompleteMultipartUploadRequest struct {
	UploadID   string `json:"UploadID"`
	ObjectName string `json:"objectName"`
	PartList   []Part `json:"partList"`
}

type Part struct {
	ETag       string `json:"eTag"`
	PartNumber int    `json:"partNumber"`
}

type UploadModelMultipart struct {
	PresignedURL string `json:"presignedURL"`
}

func (u *UploadsHTTPWrapper) UploadFile(sourcesFile string, featureFlagsWrapper FeatureFlagsWrapper) (*string, error) {
	preSignedURL, err := u.getPresignedURLForUploading()
	if err != nil {
		return nil, errors.Errorf("Failed creating pre-signed URL - %s", err.Error())
	}
	preSignedURLBytes, err := json.Marshal(*preSignedURL)
	if err != nil {
		return nil, errors.Errorf("Failed to marshal pre-signed URL - %s", err.Error())
	}
	*preSignedURL = string(preSignedURLBytes)
	viper.Set(commonParams.UploadURLEnv, *preSignedURL)

	file, err := os.Open(sourcesFile)
	if err != nil {
		return nil, errors.Errorf("Failed to open file %s: %s", sourcesFile, err.Error())
	}
	// Close the file later
	defer func() {
		_ = file.Close()
	}()

	accessToken, err := GetAccessToken()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(preSignedURLBytes, preSignedURL)
	if err != nil {
		return nil, errors.Errorf("Failed to unmarshal pre-signed URL - %s", err.Error())
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, errors.Errorf("Failed to stat file %s: %s", sourcesFile, err.Error())
	}
	flagResponse, _ := GetSpecificFeatureFlag(featureFlagsWrapper, MinioEnabled)
	useAccessToken := flagResponse.Status
	resp, err := SendHTTPRequestByFullURLContentLength(http.MethodPut, *preSignedURL, file, stat.Size(), useAccessToken, NoTimeout, accessToken, true)
	if err != nil {

		return nil, errors.Errorf("Invoking HTTP request to upload file failed - %s", err.Error())
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, errors.Errorf("%s\n%s", errorConstants.StatusUnauthorized,
			generateUploadFileFailedMessage(*preSignedURL))
	case http.StatusOK:
		return preSignedURL, nil
	default:
		return nil, errors.Errorf("response status code %d.\n%s",
			resp.StatusCode, generateUploadFileFailedMessage(*preSignedURL))
	}
}

func generateUploadFileFailedMessage(preSignedURL string) string {
	var msg string
	parsedURL, parseErr := url.Parse(preSignedURL)
	if parseErr != nil {
		msg = fmt.Sprintf(errorConstants.FailedUploadFileMsgWithURL, preSignedURL)
	} else {
		msg = fmt.Sprintf(errorConstants.FailedUploadFileMsgWithDomain, parsedURL.Host)
	}
	return msg
}

func (u *UploadsHTTPWrapper) getPresignedURLForUploading() (*string, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendPrivateHTTPRequest(http.MethodPost, u.path, nil, clientTimeout, true)
	if err != nil {
		return nil, errors.Errorf("invoking HTTP request to get pre-signed URL failed - %s", err.Error())
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, errors.Errorf("Parsing error model failed - %s", err.Error())
		}
		return nil, errors.Errorf("%d - %s", errorModel.Code, errorModel.Message)

	case http.StatusOK:
		model := UploadModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, errors.Errorf("Parsing upload model failed - %s", err.Error())
		}
		return &model.URL, nil

	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func NewUploadsHTTPWrapper(path string) UploadsWrapper {
	return &UploadsHTTPWrapper{
		path: path,
	}
}

func (u *UploadsHTTPWrapper) UploadFileInMultipart(sourcesFile string, featureFlagsWrapper FeatureFlagsWrapper) (*string, error) {
	fileInfo, _ := os.Stat(sourcesFile)

	startMultipartUploadRequest := StartMultipartUploadRequest{}
	startMultipartUploadRequest.FileSize = fileInfo.Size()
	startMultipartUploadResponse, err := startMultipartUpload(startMultipartUploadRequest)
	if err != nil {
		return nil, err
	}
	partList, err := SplitZipBySizeGB(sourcesFile)
	if err != nil {
		return nil, errors.Errorf("Failed to split ZIP file for multipart upload - %s", err.Error())
	}

	defer cleanUpTempParts(partList)

	for i, part := range partList {
		logger.PrintfIfVerbose("Part%d created at: %s", i+1, part)
	}

	completeMultipartUploadRequest := &CompleteMultipartUploadRequest{
		UploadID:   startMultipartUploadResponse.UploadID,
		ObjectName: startMultipartUploadResponse.ObjectName,
	}

	var presignedURLPart1 string

	for i, partPath := range partList {
		partNumber := i + 1

		presignedURL, err := getPresignedURLForMultipartUploading(startMultipartUploadResponse, partNumber)
		if err != nil {
			return nil, errors.Errorf("Failed to get presigned URL for part%d - %s", partNumber, err.Error())
		}

		if partNumber == 1 {
			presignedURLPart1 = presignedURL
		}

		etag, err := uploadPart(presignedURL, partPath, featureFlagsWrapper)
		if err != nil {
			return nil, errors.Errorf("Failed to upload part%d - %s", partNumber, err.Error())
		}

		completeMultipartUploadRequest.PartList = append(completeMultipartUploadRequest.PartList, Part{
			ETag:       etag,
			PartNumber: partNumber,
		})
	}

	err = completeMultipartUpload(*completeMultipartUploadRequest)
	if err != nil {
		return nil, errors.Errorf("Failed to complete multipart upload - %s", err.Error())
	}
	return &presignedURLPart1, nil
}

func startMultipartUpload(startMultipartUploadRequest StartMultipartUploadRequest) (StartMultipartUploadResponse, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	path := viper.GetString(commonParams.StartMultiPartUploadPathEnv)
	jsonBytes, err := json.Marshal(startMultipartUploadRequest)
	if err != nil {
		return StartMultipartUploadResponse{}, err
	}
	resp, err := SendHTTPRequest(http.MethodPost, path, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	if err != nil {
		return StartMultipartUploadResponse{}, err
	}
	decoder := json.NewDecoder(resp.Body)
	defer func() {
		_ = resp.Body.Close()
	}()
	switch resp.StatusCode {
	case http.StatusOK:
		startMultipartUpload := StartMultipartUploadResponse{}
		err = decoder.Decode(&startMultipartUpload)
		if err != nil {
			return StartMultipartUploadResponse{}, err
		}
		return startMultipartUpload, nil
	case http.StatusBadRequest:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return StartMultipartUploadResponse{}, err
		}
		return StartMultipartUploadResponse{}, errors.Errorf(errorModel.Message)
	case http.StatusUnauthorized:
		return StartMultipartUploadResponse{}, errors.New(errorConstants.StatusUnauthorized)
	default:
		return StartMultipartUploadResponse{}, errors.Errorf("Response status code %d", resp.StatusCode)
	}
}

func getPresignedURLForMultipartUploading(response StartMultipartUploadResponse, partNumber int) (string, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	path := viper.GetString(commonParams.MultipartPresignedPathEnv)

	multipartPresignedURL := MultipartPresignedURL{
		ObjectName: response.ObjectName,
		UploadID:   response.UploadID,
		PartNumber: partNumber,
	}
	jsonBytes, err := json.Marshal(multipartPresignedURL)
	if err != nil {
		return "", err
	}

	resp, err := SendHTTPRequest(http.MethodPost, path, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return "", err
		}
		return "", errors.Errorf("%d - %s", errorModel.Code, errorModel.Message)

	case http.StatusOK:
		model := UploadModelMultipart{}
		err = decoder.Decode(&model)
		if err != nil {
			return "", err
		}
		return model.PresignedURL, nil

	default:
		return "", errors.Errorf("Response status code %d", resp.StatusCode)
	}
}

func uploadPart(preSignedURL, sourcesFile string, featureFlagsWrapper FeatureFlagsWrapper) (string, error) {
	if preSignedURL == "" {
		return "", errors.New("PreSignedURL is empty or nil")
	}

	file, err := os.Open(sourcesFile)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	accessToken, err := GetAccessToken()
	if err != nil {
		return "", err
	}

	stat, err := file.Stat()
	if err != nil {
		return "", err
	}
	flagResponse, _ := GetSpecificFeatureFlag(featureFlagsWrapper, MinioEnabled)
	useAccessToken := flagResponse.Status
	resp, err := SendHTTPRequestByFullURLContentLength(http.MethodPut, preSignedURL, file, stat.Size(), useAccessToken, NoTimeout, accessToken, true)
	if err != nil {
		return "", err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return "", errors.Errorf(errorConstants.StatusUnauthorized)
	case http.StatusOK:
		return resp.Header.Get("Etag"), nil
	case http.StatusBadRequest:
		body, err := io.ReadAll(resp.Body)
		defer func() {
			_ = resp.Body.Close()
		}()
		if err != nil {
			return "", err
		}
		return "", errors.Errorf("Bad request while uploading part -  %s", string(body))
	default:
		return "", errors.Errorf("Response status code %d", resp.StatusCode)
	}
}

func completeMultipartUpload(completeMultipartUploadRequest CompleteMultipartUploadRequest) error {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	path := viper.GetString(commonParams.CompleteMultipartUploadPathEnv)
	jsonBytes, err := json.Marshal(completeMultipartUploadRequest)
	if err != nil {
		return err
	}
	resp, err := SendHTTPRequest(http.MethodPost, path, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(resp.Body)
	defer func() {
		_ = resp.Body.Close()
	}()
	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusUnauthorized:
		return errors.New(errorConstants.StatusUnauthorized)
	default:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return err
		}
		return errors.Errorf("%d - %s", errorModel.Code, errorModel.Message)
	}
}

func SplitZipBySizeGB(zipFilePath string) ([]string, error) {
	partSizeBytes := getPartSizeBytes()
	f, err := os.Open(zipFilePath)
	if err != nil {
		return nil, err
	}
	defer closeFileVerbose(f)

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() == 0 {
		return nil, err
	}

	partSizes := calculatePartSizes(stat.Size(), partSizeBytes)
	partNames, err := createParts(f, partSizes)
	if err != nil {
		cleanUpTempParts(partNames)
		return nil, err
	}

	return partNames, nil
}

func getPartSizeBytes() int64 {
	partChunkSizeStr := viper.GetString(commonParams.MultipartFileSizeKey)
	partChunkSizeFloat, err := strconv.ParseFloat(partChunkSizeStr, 64)
	if err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("Configured part size '%s' is invalid or empty. Defaulting to 2 GB.", partChunkSizeStr))
		partChunkSizeFloat = 2
	}
	truncatedSize := int64(partChunkSizeFloat)
	if truncatedSize < 1 || truncatedSize > 5 {
		logger.PrintIfVerbose(fmt.Sprintf("Configured part size %d GB is outside the allowed range (1 – 5 GB). Defaulting to 2 GB.", truncatedSize))
		truncatedSize = 2
	}
	logger.PrintIfVerbose("Splitting zip file into parts of size: " + fmt.Sprintf("%.0f", float64(truncatedSize)) + " GB")
	const bytesPerGB = 1024 * 1024 * 1024
	return int64(float64(truncatedSize) * float64(bytesPerGB))
}

func calculatePartSizes(totalSize, partSizeBytes int64) []int64 {
	numParts := int(totalSize / partSizeBytes)
	if totalSize%partSizeBytes != 0 {
		numParts++
	}
	partSizes := make([]int64, numParts)
	for i := 0; i < numParts; i++ {
		remaining := totalSize - int64(i)*partSizeBytes
		if remaining >= partSizeBytes {
			partSizes[i] = partSizeBytes
		} else {
			partSizes[i] = remaining
		}
	}
	return partSizes
}

func createParts(f *os.File, partSizes []int64) ([]string, error) {
	partNames := make([]string, len(partSizes))
	for i, size := range partSizes {
		partFile, err := os.CreateTemp("", fmt.Sprintf("cx-part%d-*", i+1))
		if err != nil {
			return partNames, err
		}
		offset := int64(0)
		for j := 0; j < i; j++ {
			offset += partSizes[j]
		}
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			err := partFile.Close()
			if err != nil {
				return nil, err
			}
			err = os.Remove(partFile.Name())
			if err != nil {
				return nil, err
			}
			return partNames, err
		}
		if _, err := io.CopyN(partFile, f, size); err != nil && err != io.EOF {
			err := partFile.Close()
			if err != nil {
				return nil, err
			}
			err = os.Remove(partFile.Name())
			if err != nil {
				return nil, err
			}
			return partNames, err
		}
		if err := partFile.Sync(); err != nil {
			return partNames, err
		}
		if err := partFile.Close(); err != nil {
			return partNames, err
		}
		partNames[i] = partFile.Name()
	}
	return partNames, nil
}

func closeFileVerbose(f *os.File) {
	if err := f.Close(); err != nil {
		logger.PrintfIfVerbose("Warning: failed to close input file - %v", err)
	}
}

// cleanUpTempParts removes the temporary part files created during multipart upload.
func cleanUpTempParts(partList []string) {
	cleanupMaxRetries := 3
	for i, partPath := range partList {
		if partPath != "" {
			logger.PrintIfVerbose(fmt.Sprintf("Cleaning up temporary part%d - %s", i+1, partPath))
			tries := cleanupMaxRetries
			for attempt := 1; tries > 0; attempt++ {
				removeErr := os.Remove(partPath)
				if removeErr != nil {
					if os.IsNotExist(removeErr) {
						logger.PrintIfVerbose(fmt.Sprintf("Temporary part%d already removed - %s", i+1, partPath))
						break
					}
					logger.PrintIfVerbose(fmt.Sprintf(
						"Failed to remove temporary part%d - Attempt %d/%d - %v",
						i+1,
						attempt,
						cleanupMaxRetries,
						removeErr,
					))
					tries--
					Wait(attempt)
				} else {
					logger.PrintIfVerbose(fmt.Sprintf("Removed temporary part%d", i+1))
					break
				}
			}
			if tries == 0 {
				logger.PrintIfVerbose(fmt.Sprintf("Failed to remove temporary part%d - %s", i+1, partPath))
			}
		} else {
			logger.PrintIfVerbose(fmt.Sprintf("No temporary part%d to clean", i+1))
		}
	}
}

// Wait implements exponential backoff wait strategy
func Wait(attempt int) {
	cleanupRetryWaitSeconds := 15
	// Calculate exponential backoff delay
	waitDuration := time.Duration(cleanupRetryWaitSeconds * (1 << (attempt - 1))) // 2^(attempt-1)
	logger.PrintIfVerbose(fmt.Sprintf("Waiting %d seconds before retrying...", waitDuration))
	time.Sleep(waitDuration * time.Second)
}
