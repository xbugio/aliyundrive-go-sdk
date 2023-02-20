package aliyundrive

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"time"
)

// 最顶部的根文件Id
const RootFileId = "root"

// List接口可接受的最大limit参数
const LimitMax = 200

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * GB
)

const (
	// 降序排序
	OrderDirectionDesc = "DESC"
	// 升序排序
	OrderDirectionAsc = "ASC"
	// 按名字排序
	OrderByName = "name"
	// 按更新时间排序
	OrderByUpdatedAt = "updated_at"
	// 按创建时间排序
	OrderByCreatedAt = "created_at"
)

// 计算秒传接口proof值的数据的起始位置
//
// 秒传接口需要传proof值，proof值是文件某一处位置开始往后取8个字节的数据，然后进行base64的字符串
// 数据开始的位置是根据accesstoken和文件大小通过算法计算得到的
func GetProofStart(accessToken string, size uint64) uint64 {
	hash := md5.Sum([]byte(accessToken))
	bigInt := new(big.Int).SetBytes(hash[:8])
	return new(big.Int).Mod(bigInt, new(big.Int).SetUint64(size)).Uint64()
}

// 文件/目录的完整结构体
//
// 由于云盘各个接口返回的文件或目录的字段都有很多重叠但又有差异，
// 所以设计全部放在一个结构体内，在请求不同的接口或者不同文件目录类型、格式的时候，
// 部分特别的字段可能会出现或者没有，需要调用者自己关注测试好
type Item struct {
	FileId          string    `json:"file_id"`
	Name            string    `json:"name"`
	ParentFileId    string    `json:"parent_file_id"`
	Type            string    `json:"type"`
	Starred         bool      `json:"starred"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	TrashedAt       time.Time `json:"trashed_at"`
	GMTExpired      time.Time `json:"gmt_expired"`
	Category        string    `json:"category"`
	ContentHash     string    `json:"content_hash"`
	Size            uint64    `json:"size"`
	UserMeta        string    `json:"user_meta"`
	FileExtension   string    `json:"file_extension"`
	MimeType        string    `json:"mime_type"`
	PunishFlag      int       `json:"punish_flag"`
	Thumbnail       string    `json:"thumbnail"`
	Url             string    `json:"url"`
	Hidden          bool      `json:"hidden"`
	Trashed         bool      `json:"trashed"`
	Status          string    `json:"status"`
	EncryptMode     string    `json:"encrypt_mode"`
	ContentHashName string    `json:"content_hash_name"`
	ContentType     string    `json:"content_type"`
	Crc64Hash       string    `json:"crc64_hash"`
	MimeExtension   string    `json:"mime_extension"`
	DownloadUrl     string    `json:"download_url"`
	UploadId        string    `json:"upload_id"`
	Labels          []string  `json:"labels"`
}

// Item检索器，根据不同的字段条件找对应的item
type ItemQuery []*Item

// 根据name字段查找item
func (q ItemQuery) ByName(name string) (item *Item, exists bool) {
	for _, i := range q {
		if i.Name == name {
			item = i
			exists = true
			break
		}
	}
	return
}

type GetPersonalInfoRequest struct {
}

type GetPersonalInfoResponse struct {
	PersonalRightsInfo *struct {
		Name       string `json:"name"`
		SpuId      string `json:"spu_id"`
		IsExpires  bool   `json:"is_expires"`
		Privileges []*struct {
			FeatureId     string `json:"feature_id"`
			FeatureAttrId string `json:"feature_attr_id"`
			Quota         int    `json:"quota"`
		} `json:"privileges"`
	} `json:"personal_rights_info"`
	PersonalSpaceInfo *struct {
		TotalSize uint64 `json:"total_size"`
		UsedSize  uint64 `json:"used_size"`
	} `json:"personal_space_info"`
}

// 获取用户个人及网盘信息接口
func (c *Drive) DoGetPersonalInfoRequest(ctx context.Context, request GetPersonalInfoRequest) (*GetPersonalInfoResponse, error) {
	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/v2/databox/get_personal_info", Object{})
	if err != nil {
		return nil, err
	}

	result := new(GetPersonalInfoResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type ListRequest struct {
	// 需要获取列表的目录文件Id，必须
	ParentFileId string `json:"parent_file_id,omitempty"`
	// 排序字段，可选
	OrderBy string `json:"order_by,omitempty"`
	// 排序方式，升序/降序，可选
	OrderDirection string `json:"order_direction,omitempty"`
	// 最大返回条目，不能超过LimitMax，可选
	Limit int `json:"limit,omitempty"`
	// 分页标记，可选
	NextMarker string `json:"marker,omitempty"`
}
type ListResponse struct {
	// 列出的目录文件
	Items []*Item `json:"items"`
	// 下一页分页标记，为空则表示没有更多数据了
	NextMarker string `json:"next_marker"`
}

// 获取目录下文件列表接口
func (c *Drive) DoListRequest(ctx context.Context, request ListRequest) (*ListResponse, error) {
	params := &struct {
		DriveId string `json:"drive_id"`
		Fields  string `json:"fields"`
		ListRequest
	}{
		DriveId:     c.driveId,
		Fields:      "*",
		ListRequest: request,
	}
	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/adrive/v3/file/list", params)
	if err != nil {
		return nil, err
	}

	result := new(ListResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type SearchRequest struct {
	// 搜索文件目录关键词，必须
	Name string `json:"-"`
	// 排序字段，可选
	OrderBy string `json:"order_by,omitempty"`
	// 排序方式，升序/降序，可选
	OrderDirection string `json:"order_direction,omitempty"`
	// 最大返回条目，不能超过LimitMax，可选
	Limit int `json:"limit,omitempty"`
	// 分页标记，可选
	NextMarker string `json:"marker,omitempty"`
}

type SearchResponse struct {
	// 搜索到的文件目录
	Items []*Item `json:"items"`
	// 下一页分页标记，为空则表示没有更多数据了
	NextMarker string `json:"next_marker"`
}

// 搜索文件目录接口
func (c *Drive) DoSearchRequest(ctx context.Context, request SearchRequest) (*SearchResponse, error) {

	params := &struct {
		DriveId string `json:"drive_id"`
		Query   string `json:"query"`
		SearchRequest
	}{
		DriveId:       c.driveId,
		SearchRequest: request,
	}
	params.Query = `name match "` + params.Name + `"`

	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/adrive/v3/file/search", params)
	if err != nil {
		return nil, err
	}

	result := new(SearchResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type GetRequest struct {
	// 文件Id，必须
	FileId string `json:"file_id"`
}

type GetResponse struct {
	Item
}

// 获取文件详细信息接口
func (c *Drive) DoGetRequest(ctx context.Context, request GetRequest) (*GetResponse, error) {

	params := &struct {
		DriveId string `json:"drive_id"`
		GetRequest
	}{
		DriveId:    c.driveId,
		GetRequest: request,
	}
	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/v2/file/get", params)
	if err != nil {
		return nil, err
	}

	result := new(GetResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type GetDownloadUrlRequest struct {
	// 文件Id，必须
	FileId string `json:"file_id"`
}

type GetDownloadUrlResponse struct {
	// 文件Id
	FileId string `json:"file_id"`
	// 文件大小
	Size uint64 `json:"size"`
	// 文件Hash
	ContentHash string `json:"content_hash"`
	// 文件Hash算法
	ContentHashName string `json:"content_hash_name"`
	// CRC Hash
	Crc64Hash string `json:"crc64_hash"`
	// 地址过期时间
	Expiration time.Time `json:"expiration"`
	// 内部用下载地址
	InternalUrl string `json:"internal_url"`
	// 下载地址
	Url string `json:"url"`
}

// 获取文件下载链接接口
func (c *Drive) DoGetDownloadUrlRequest(ctx context.Context, request GetDownloadUrlRequest) (*GetDownloadUrlResponse, error) {

	params := &struct {
		DriveId string `json:"drive_id"`
		GetDownloadUrlRequest
	}{
		DriveId:               c.driveId,
		GetDownloadUrlRequest: request,
	}
	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/v2/file/get_download_url", params)
	if err != nil {
		return nil, err
	}

	result := new(GetDownloadUrlResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type GetFolderSizeInfoRequest struct {
	// 文件Id，必须
	FileId string `json:"file_id"`
}

type GetFolderSizeInfoResponse struct {
	// 文件数量
	FileCount uint64 `json:"file_count"`
	// 文件夹数量
	FolderCount uint64 `json:"folder_count"`
	// 使用空间（有些不准）
	Size uint64 `json:"size"`
}

// 获取目录信息接口
func (c *Drive) DoGetFolderSizeInfoRequest(ctx context.Context, request GetFolderSizeInfoRequest) (*GetFolderSizeInfoResponse, error) {

	params := &struct {
		DriveId string `json:"drive_id"`
		GetFolderSizeInfoRequest
	}{
		DriveId:                  c.driveId,
		GetFolderSizeInfoRequest: request,
	}
	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/adrive/v1/file/get_folder_size_info", params)
	if err != nil {
		return nil, err
	}

	result := new(GetFolderSizeInfoResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type CreateFolderRequest struct {
	// 名称，必须
	Name string `json:"name"`
	// 父级目录文件Id，必须
	ParentFileId string `json:"parent_file_id"`
}

type CreateFolderResponse struct {
	// 文件Id
	FileId string `json:"file_id"`
	// 文件名
	FileName string `json:"file_name"`
	// 父级目录文件Id
	ParentFileId string `json:"parent_file_id"`
	// 类型
	Type        string `json:"type"`
	EncryptMode string `json:"encrypt_mode"`
}

// 创建目录接口
func (c *Drive) DoCreateFolderRequest(ctx context.Context, request CreateFolderRequest) (*CreateFolderResponse, error) {
	params := &struct {
		DriveId       string `json:"drive_id"`
		CheckNameMode string `json:"check_name_mode"`
		Type          string `json:"type"`
		CreateFolderRequest
	}{
		DriveId:             c.driveId,
		CheckNameMode:       "refuse",
		Type:                "folder",
		CreateFolderRequest: request,
	}

	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/adrive/v2/file/createWithFolders", params)
	if err != nil {
		return nil, err
	}

	result := new(CreateFolderResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type CreateFileRequest struct {
	// 文件名，必须
	Name string `json:"name"`
	// 父级目录文件Id，必须
	ParentFileId string `json:"parent_file_id"`
	// 文件大小，必须
	Size uint64 `json:"size"`
	// 文件前1kb的数据的sha1值，可选，空则不考虑秒传
	PreHash string `json:"pre_hash"`
	// 文件分片大小，必须
	ChunkSize uint64 `json:"-"`
}

type CreateFileResponse struct {
	// 文件Id
	FileId string `json:"file_id"`
	// 文件名
	FileName string `json:"file_name"`
	// 父级目录文件Id
	ParentFileId string `json:"parent_file_id"`
	// 是否秒传
	RapidUpload bool `json:"rapid_upload"`
	// 文件类型
	Type        string `json:"type"`
	EncryptMode string `json:"encrypt_mode"`
	// 上传Id
	UploadId string `json:"upload_id"`
	// 分片信息
	PartInfoList []*struct {
		// 分片编号
		PartNumber  int    `json:"part_number"`
		ContentType string `json:"content_type"`
		// 分片内部上传地址
		InternalUploadUrl string `json:"internal_upload_url"`
		// 分片上传地址
		UploadUrl string `json:"upload_url"`
	} `json:"part_info_list"`
}

// 创建文件接口
//
// 若error返回的是PreHashMatched，表明可以尝试秒传上传
func (c *Drive) DoCreateFileRequest(ctx context.Context, request CreateFileRequest) (*CreateFileResponse, error) {
	params := &struct {
		DriveId       string `json:"drive_id"`
		DeviceName    string `json:"device_name"`
		CreateScene   string `json:"create_scene"`
		CheckNameMode string `json:"check_name_mode"`
		Type          string `json:"type"`
		PartInfoList  Array  `json:"part_info_list"`
		CreateFileRequest
	}{
		DriveId:           c.driveId,
		CheckNameMode:     "auto_rename",
		CreateScene:       "file_upload",
		Type:              "file",
		CreateFileRequest: request,
	}

	var partCount int
	if params.ChunkSize == 0 {
		partCount = 1
	} else {
		partCount = int(params.Size / params.ChunkSize)
		if params.Size%params.ChunkSize > 0 {
			partCount++
		}
	}

	params.PartInfoList = make(Array, partCount)
	for i := 0; i < partCount; i++ {
		params.PartInfoList[i] = Object{"part_number": i}
	}

	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/adrive/v2/file/createWithFolders", params)
	if err != nil {
		return nil, err
	}

	result := new(CreateFileResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type DownloadFileRequest struct {
	// 云盘文件下载地址，必须
	Url string
	// 额外的请求头，比如Range，可选
	Header http.Header
}

type DownloadFileResponse struct {
	// 文件流，需要读取完且关闭
	Reader io.ReadCloser
}

// 下载文件数据
func (c *Drive) DoDownloadFileRequest(ctx context.Context, request DownloadFileRequest) (*DownloadFileResponse, error) {
	httpRequest, err := http.NewRequestWithContext(ctx, "GET", request.Url, nil)
	if err != nil {
		return nil, err
	}
	for k := range request.Header {
		httpRequest.Header.Add(k, request.Header.Get(k))
	}
	httpRequest.Header.Set("Origin", "https://www.aliyundrive.com")
	httpRequest.Header.Set("Referer", "https://www.aliyundrive.com/")

	resp, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	return &DownloadFileResponse{
		Reader: resp.Body,
	}, nil
}

type UploadFileRequest struct {
	// 云盘文件上传地址，必须
	Url string
	// 上传数据流，必须
	File io.Reader
}

type UploadFileResponse struct {
}

// 上传文件数据
func (c *Drive) DoUploadFileRequest(ctx context.Context, request UploadFileRequest) (*UploadFileResponse, error) {
	httpRequest, err := http.NewRequestWithContext(ctx, "PUT", request.Url, request.File)
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Origin", "https://www.aliyundrive.com")
	httpRequest.Header.Set("Referer", "https://www.aliyundrive.com/")

	resp, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return &UploadFileResponse{}, nil
}

type CompleteUploadFileRequest struct {
	// 文件Id，必须
	FileId string `json:"file_id"`
	// 上传Id，必须
	UploadId string `json:"upload_id"`
}

type CompleteUploadFileResponse struct {
	Item
}

// 完成文件的分片上传接口
func (c *Drive) DoCompleteUploadFileRequest(ctx context.Context, request CompleteUploadFileRequest) (*CompleteUploadFileResponse, error) {
	params := &struct {
		DriveId string `json:"drive_id"`
		CompleteUploadFileRequest
	}{
		DriveId:                   c.driveId,
		CompleteUploadFileRequest: request,
	}

	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/v2/file/complete", params)
	if err != nil {
		return nil, err
	}

	result := new(CompleteUploadFileResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type RapidCreateFileRequest struct {
	// 文件名，必须
	Name string `json:"name"`
	// 父级目录文件Id，必须
	ParentFileId string `json:"parent_file_id"`
	// 文件大小，必须
	Size uint64 `json:"size"`
	// 文件分片大小，必须
	ChunkSize uint64 `json:"-"`
	// 文件SHA1，必须
	ContentHash string `json:"content_hash"`
	// 文件某个位置开始读取8字节后base64的值，开始位置用GetProofStart计算，必须
	ProofCode string `json:"proof_code"`

	// 本次创建使用的accesstoken，必须
	//
	// 因为accesstoken与proofcode相关，
	// 所以无法再使用内部token管理器内的accesstoken，
	// 需要人为指定参与计算proofcode的accesstoken
	AccessToken string `json:"-"`
}

type RapidCreateFileResponse struct {
	// 文件Id
	FileId string `json:"file_id"`
	// 文件名
	FileName string `json:"file_name"`
	// 父级目录文件Id
	ParentFileId string `json:"parent_file_id"`
	// 是否秒传
	RapidUpload bool `json:"rapid_upload"`
	// 文件类型
	Type        string `json:"type"`
	EncryptMode string `json:"encrypt_mode"`
	// 上传Id
	UploadId string `json:"upload_id"`
}

// 秒传文件接口
//
// 在创建文件接口返回PreHashMatched错误后，可调用该接口秒传文件
func (c *Drive) DoRapidCreateFileRequest(ctx context.Context, request RapidCreateFileRequest) (*RapidCreateFileResponse, error) {
	params := &struct {
		DriveId         string `json:"drive_id"`
		DeviceName      string `json:"device_name"`
		CreateScene     string `json:"create_scene"`
		CheckNameMode   string `json:"check_name_mode"`
		ContentHashName string `json:"content_hash_name"`
		Type            string `json:"type"`
		ProofVersion    string `json:"proof_version"`
		PartInfoList    Array  `json:"part_info_list"`
		RapidCreateFileRequest
	}{
		DriveId:                c.driveId,
		CheckNameMode:          "auto_rename",
		CreateScene:            "file_upload",
		ContentHashName:        "sha1",
		Type:                   "file",
		ProofVersion:           "v1",
		RapidCreateFileRequest: request,
	}

	var partCount int
	if params.ChunkSize == 0 {
		partCount = 1
	} else {
		partCount = int(params.Size / params.ChunkSize)
		if params.Size%params.ChunkSize > 0 {
			partCount++
		}
	}

	params.PartInfoList = make(Array, partCount)
	for i := 0; i < partCount; i++ {
		params.PartInfoList[i] = Object{"part_number": i}
	}

	httpRequest, err := c.toRequest(ctx, "https://api.aliyundrive.com/adrive/v2/file/createWithFolders", params)
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+params.AccessToken)
	resp, err := c.doRequest(httpRequest)
	if err != nil {
		return nil, err
	}

	result := new(RapidCreateFileResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type RenameRequest struct {
	// 文件Id，必须
	FileId string `json:"file_id"`
	// 文件名，必须
	Name string `json:"name"`
}

type RenameResponse struct {
	Item
}

// 重命名文件接口
func (c *Drive) DoRenameRequest(ctx context.Context, request RenameRequest) (*RenameResponse, error) {
	params := &struct {
		DriveId       string `json:"drive_id"`
		CheckNameMode string `json:"check_name_mode"`
		RenameRequest
	}{
		DriveId:       c.driveId,
		CheckNameMode: "refuse",
		RenameRequest: request,
	}

	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/v3/file/update", params)
	if err != nil {
		return nil, err
	}

	result := new(RenameResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type MoveRequest struct {
	// 文件Id，必须
	FileId string `json:"file_id"`
	// 目的目录文件Id，必须
	ToParentFileId string `json:"to_parent_file_id"`
}

type MoveResponse struct {
	FileId string `json:"file_id"`
}

// 移动文件/目录接口
func (c *Drive) DoMoveRequest(ctx context.Context, request MoveRequest) (*MoveResponse, error) {
	params := &struct {
		DriveId   string `json:"drive_id"`
		ToDriveId string `json:"to_drive_id"`
		MoveRequest
	}{
		DriveId:     c.driveId,
		ToDriveId:   c.driveId,
		MoveRequest: request,
	}

	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/v3/file/move", params)
	if err != nil {
		return nil, err
	}

	result := new(MoveResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type TrashRequest struct {
	// 文件Id，必须
	FileId string `json:"file_id"`
}

type TrashResponse struct {
	// 异步任务Id
	AsyncTaskId string `json:"async_task_id"`
	// 文件Id
	FileId string `json:"file_id"`
}

// 文件/目录移到回收站接口
func (c *Drive) DoTrashRequest(ctx context.Context, request TrashRequest) (*TrashResponse, error) {
	params := &struct {
		DriveId string `json:"drive_id"`
		TrashRequest
	}{
		DriveId:      c.driveId,
		TrashRequest: request,
	}

	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/v2/recyclebin/trash", params)
	if err != nil {
		return nil, err
	}

	result := new(TrashResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type ClearTrashRequest struct{}

type ClearTrashResponse struct {
	// 异步任务Id
	AsyncTaskId string `json:"async_task_id"`
	// 任务Id
	TaskId string `json:"task_id"`
}

// 清空回收站接口
func (c *Drive) DoClearTrashRequest(ctx context.Context, request ClearTrashRequest) (*ClearTrashResponse, error) {
	params := &struct {
		DriveId string `json:"drive_id"`
		ClearTrashRequest
	}{
		DriveId:           c.driveId,
		ClearTrashRequest: request,
	}

	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/v2/recyclebin/clear", params)
	if err != nil {
		return nil, err
	}

	result := new(ClearTrashResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type ListTrashRequest struct {
	// 排序字段，可选
	OrderBy string `json:"order_by,omitempty"`
	// 排序方式，升序/降序，可选
	OrderDirection string `json:"order_direction,omitempty"`
	// 最大返回条目，不能超过LimitMax，可选
	Limit int `json:"limit,omitempty"`
	// 分页标记，可选
	NextMarker string `json:"marker,omitempty"`
}

type ListTrashResponse struct {
	// 列出文件/目录列表
	Items []*Item `json:"items"`
	// 下一页分页标记，为空则表示没有更多数据了
	NextMarker string `json:"next_marker"`
}

// 列出回收站内文件/目录接口
func (c *Drive) DoListTrashRequest(ctx context.Context, request ListTrashRequest) (*ListTrashResponse, error) {
	params := &struct {
		DriveId string `json:"drive_id"`
		ListTrashRequest
	}{
		DriveId:          c.driveId,
		ListTrashRequest: request,
	}

	resp, err := c.requestWithCredit(ctx, "https://api.aliyundrive.com/adrive/v2/recyclebin/list", params)
	if err != nil {
		return nil, err
	}

	result := new(ListTrashResponse)
	err = json.Unmarshal(resp, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type RestoreRequest struct {
	// 文件Id，必须
	FileId string `json:"file_id"`
}

type RestoreResponse struct {
}

// 恢复回收站文件/目录接口
func (c *Drive) DoRestoreRequest(ctx context.Context, request RestoreRequest) (*RestoreResponse, error) {
	params := &struct {
		DriveId string `json:"drive_id"`
		RestoreRequest
	}{
		DriveId:        c.driveId,
		RestoreRequest: request,
	}

	accessToken, err := c.tokenManager.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	httpRequest, err := c.toRequest(ctx, "https://api.aliyundrive.com/v2/recyclebin/restore", params)
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return &RestoreResponse{}, nil
}

type DeleteRequest struct {
	// 文件Id，必须
	FileId string `json:"file_id"`
}

type DeleteResponse struct {
}

// 永久删除文件/目录接口（不管是否在回收站）
func (c *Drive) DoDeleteRequest(ctx context.Context, request DeleteRequest) (*DeleteResponse, error) {
	params := &struct {
		DriveId string `json:"drive_id"`
		DeleteRequest
	}{
		DriveId:       c.driveId,
		DeleteRequest: request,
	}

	accessToken, err := c.tokenManager.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	httpRequest, err := c.toRequest(ctx, "https://api.aliyundrive.com/v3/file/delete", params)
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.HttpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return &DeleteResponse{}, nil
}
