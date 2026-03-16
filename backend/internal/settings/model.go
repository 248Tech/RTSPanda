package settings

type AppSettings struct {
	OpenAIEnabled               bool   `json:"openai_enabled"`
	OpenAIModel                 string `json:"openai_model"`
	OpenAIAPIKeySet             bool   `json:"openai_api_key_set"`
	VideoStorageEnabled         bool   `json:"video_storage_enabled"`
	VideoStorageProvider        string `json:"video_storage_provider"`
	VideoStorageLocalPath       string `json:"video_storage_local_path"`
	VideoStorageRemoteName      string `json:"video_storage_remote_name"`
	VideoStorageRemotePath      string `json:"video_storage_remote_path"`
	VideoStorageSyncIntervalSec int    `json:"video_storage_sync_interval_seconds"`
	VideoStorageMinFileAgeSec   int    `json:"video_storage_min_file_age_seconds"`
}

type OpenAIConfig struct {
	Enabled bool
	APIKey  string
	Model   string
}

type VideoStorageConfig struct {
	Enabled         bool
	Provider        string
	LocalPath       string
	RemoteName      string
	RemotePath      string
	SyncIntervalSec int
	MinFileAgeSec   int
}

type UpdateInput struct {
	OpenAIEnabled            *bool   `json:"openai_enabled"`
	OpenAIModel              *string `json:"openai_model"`
	OpenAIAPIKey             *string `json:"openai_api_key"`
	ClearOpenAIAPIKey        *bool   `json:"clear_openai_api_key"`
	VideoStorageEnabled      *bool   `json:"video_storage_enabled"`
	VideoStorageProvider     *string `json:"video_storage_provider"`
	VideoStorageLocalPath    *string `json:"video_storage_local_path"`
	VideoStorageRemoteName   *string `json:"video_storage_remote_name"`
	VideoStorageRemotePath   *string `json:"video_storage_remote_path"`
	VideoStorageSyncInterval *int    `json:"video_storage_sync_interval_seconds"`
	VideoStorageMinFileAge   *int    `json:"video_storage_min_file_age_seconds"`
}
