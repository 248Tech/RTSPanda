package settings

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalid = errors.New("invalid input")

const (
	keyOpenAIEnabled = "openai_enabled"
	keyOpenAIModel   = "openai_model"
	keyOpenAIAPIKey  = "openai_api_key"

	keyVideoStorageEnabled      = "video_storage_enabled"
	keyVideoStorageProvider     = "video_storage_provider"
	keyVideoStorageLocalPath    = "video_storage_local_path"
	keyVideoStorageRemoteName   = "video_storage_remote_name"
	keyVideoStorageRemotePath   = "video_storage_remote_path"
	keyVideoStorageSyncInterval = "video_storage_sync_interval_seconds"
	keyVideoStorageMinFileAge   = "video_storage_min_file_age_seconds"

	defaultOpenAIModel                 = "gpt-4o-mini"
	defaultVideoStorageProvider        = "local_server"
	defaultVideoStorageRemotePath      = "RTSPanda"
	defaultVideoStorageSyncIntervalSec = 300
	defaultVideoStorageMinFileAgeSec   = 120
	minVideoStorageSyncIntervalSec     = 30
	minVideoStorageMinFileAgeSec       = 15
	maxVideoStorageSyncIntervalSec     = 86400
	maxVideoStorageMinFileAgeSec       = 3600
	videoStorageProviderLocalServer    = "local_server"
	videoStorageProviderDropbox        = "dropbox"
	videoStorageProviderGoogleDrive    = "google_drive"
	videoStorageProviderOneDrive       = "onedrive"
	videoStorageProviderProtonDrive    = "proton_drive"
)

var validVideoStorageProviders = map[string]struct{}{
	videoStorageProviderLocalServer: {},
	videoStorageProviderDropbox:     {},
	videoStorageProviderGoogleDrive: {},
	videoStorageProviderOneDrive:    {},
	videoStorageProviderProtonDrive: {},
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get() (AppSettings, error) {
	openaiCfg, err := s.GetOpenAIConfig()
	if err != nil {
		return AppSettings{}, err
	}
	videoCfg, err := s.GetVideoStorageConfig()
	if err != nil {
		return AppSettings{}, err
	}

	return AppSettings{
		OpenAIEnabled:               openaiCfg.Enabled,
		OpenAIModel:                 openaiCfg.Model,
		OpenAIAPIKeySet:             strings.TrimSpace(openaiCfg.APIKey) != "",
		VideoStorageEnabled:         videoCfg.Enabled,
		VideoStorageProvider:        videoCfg.Provider,
		VideoStorageLocalPath:       videoCfg.LocalPath,
		VideoStorageRemoteName:      videoCfg.RemoteName,
		VideoStorageRemotePath:      videoCfg.RemotePath,
		VideoStorageSyncIntervalSec: videoCfg.SyncIntervalSec,
		VideoStorageMinFileAgeSec:   videoCfg.MinFileAgeSec,
	}, nil
}

func (s *Service) GetOpenAIConfig() (OpenAIConfig, error) {
	enabledRaw, enabledExists, err := s.repo.Get(keyOpenAIEnabled)
	if err != nil {
		return OpenAIConfig{}, err
	}
	modelRaw, modelExists, err := s.repo.Get(keyOpenAIModel)
	if err != nil {
		return OpenAIConfig{}, err
	}
	apiKeyRaw, _, err := s.repo.Get(keyOpenAIAPIKey)
	if err != nil {
		return OpenAIConfig{}, err
	}

	enabled := false
	if enabledExists {
		enabled = parseBoolSetting(enabledRaw, false)
	}

	model := defaultOpenAIModel
	if modelExists {
		model = strings.TrimSpace(modelRaw)
	}
	if model == "" {
		model = defaultOpenAIModel
	}

	return OpenAIConfig{
		Enabled: enabled,
		APIKey:  strings.TrimSpace(apiKeyRaw),
		Model:   model,
	}, nil
}

func (s *Service) GetVideoStorageConfig() (VideoStorageConfig, error) {
	enabledRaw, enabledExists, err := s.repo.Get(keyVideoStorageEnabled)
	if err != nil {
		return VideoStorageConfig{}, err
	}
	providerRaw, providerExists, err := s.repo.Get(keyVideoStorageProvider)
	if err != nil {
		return VideoStorageConfig{}, err
	}
	localPathRaw, _, err := s.repo.Get(keyVideoStorageLocalPath)
	if err != nil {
		return VideoStorageConfig{}, err
	}
	remoteNameRaw, _, err := s.repo.Get(keyVideoStorageRemoteName)
	if err != nil {
		return VideoStorageConfig{}, err
	}
	remotePathRaw, remotePathExists, err := s.repo.Get(keyVideoStorageRemotePath)
	if err != nil {
		return VideoStorageConfig{}, err
	}
	syncIntervalRaw, syncIntervalExists, err := s.repo.Get(keyVideoStorageSyncInterval)
	if err != nil {
		return VideoStorageConfig{}, err
	}
	minAgeRaw, minAgeExists, err := s.repo.Get(keyVideoStorageMinFileAge)
	if err != nil {
		return VideoStorageConfig{}, err
	}

	enabled := false
	if enabledExists {
		enabled = parseBoolSetting(enabledRaw, false)
	}

	provider := defaultVideoStorageProvider
	if providerExists {
		provider = normalizeVideoStorageProvider(providerRaw)
	}
	if provider == "" {
		provider = defaultVideoStorageProvider
	}
	if !isValidVideoStorageProvider(provider) {
		provider = defaultVideoStorageProvider
	}

	remotePath := defaultVideoStorageRemotePath
	if remotePathExists {
		remotePath = normalizeRemotePath(remotePathRaw)
	}
	if remotePath == "" {
		remotePath = defaultVideoStorageRemotePath
	}

	syncIntervalSec := defaultVideoStorageSyncIntervalSec
	if syncIntervalExists {
		syncIntervalSec = parseIntSetting(syncIntervalRaw, defaultVideoStorageSyncIntervalSec)
	}
	if syncIntervalSec < minVideoStorageSyncIntervalSec {
		syncIntervalSec = minVideoStorageSyncIntervalSec
	}
	if syncIntervalSec > maxVideoStorageSyncIntervalSec {
		syncIntervalSec = maxVideoStorageSyncIntervalSec
	}

	minFileAgeSec := defaultVideoStorageMinFileAgeSec
	if minAgeExists {
		minFileAgeSec = parseIntSetting(minAgeRaw, defaultVideoStorageMinFileAgeSec)
	}
	if minFileAgeSec < minVideoStorageMinFileAgeSec {
		minFileAgeSec = minVideoStorageMinFileAgeSec
	}
	if minFileAgeSec > maxVideoStorageMinFileAgeSec {
		minFileAgeSec = maxVideoStorageMinFileAgeSec
	}

	return VideoStorageConfig{
		Enabled:         enabled,
		Provider:        provider,
		LocalPath:       strings.TrimSpace(localPathRaw),
		RemoteName:      strings.TrimSpace(remoteNameRaw),
		RemotePath:      remotePath,
		SyncIntervalSec: syncIntervalSec,
		MinFileAgeSec:   minFileAgeSec,
	}, nil
}

func (s *Service) Update(input UpdateInput) (AppSettings, error) {
	currentOpenAI, err := s.GetOpenAIConfig()
	if err != nil {
		return AppSettings{}, err
	}
	currentVideoStorage, err := s.GetVideoStorageConfig()
	if err != nil {
		return AppSettings{}, err
	}

	openaiEnabled := currentOpenAI.Enabled
	if input.OpenAIEnabled != nil {
		openaiEnabled = *input.OpenAIEnabled
	}

	openaiModel := currentOpenAI.Model
	if strings.TrimSpace(openaiModel) == "" {
		openaiModel = defaultOpenAIModel
	}
	if input.OpenAIModel != nil {
		openaiModel = strings.TrimSpace(*input.OpenAIModel)
		if openaiModel == "" {
			return AppSettings{}, fmt.Errorf("%w: openai_model is required", ErrInvalid)
		}
	}

	openaiAPIKey := currentOpenAI.APIKey
	if input.OpenAIAPIKey != nil {
		next := strings.TrimSpace(*input.OpenAIAPIKey)
		if next == "" {
			return AppSettings{}, fmt.Errorf("%w: openai_api_key cannot be empty; omit to keep existing or use clear_openai_api_key", ErrInvalid)
		}
		openaiAPIKey = next
	}

	clearOpenAIKey := false
	if input.ClearOpenAIAPIKey != nil {
		clearOpenAIKey = *input.ClearOpenAIAPIKey
	}
	if clearOpenAIKey {
		openaiAPIKey = ""
	}

	videoStorageEnabled := currentVideoStorage.Enabled
	if input.VideoStorageEnabled != nil {
		videoStorageEnabled = *input.VideoStorageEnabled
	}

	videoStorageProvider := currentVideoStorage.Provider
	if input.VideoStorageProvider != nil {
		videoStorageProvider = normalizeVideoStorageProvider(*input.VideoStorageProvider)
	}
	if videoStorageProvider == "" {
		videoStorageProvider = defaultVideoStorageProvider
	}
	if !isValidVideoStorageProvider(videoStorageProvider) {
		return AppSettings{}, fmt.Errorf("%w: unsupported video_storage_provider %q", ErrInvalid, videoStorageProvider)
	}

	videoStorageLocalPath := strings.TrimSpace(currentVideoStorage.LocalPath)
	if input.VideoStorageLocalPath != nil {
		videoStorageLocalPath = strings.TrimSpace(*input.VideoStorageLocalPath)
	}

	videoStorageRemoteName := strings.TrimSpace(currentVideoStorage.RemoteName)
	if input.VideoStorageRemoteName != nil {
		videoStorageRemoteName = strings.TrimSpace(*input.VideoStorageRemoteName)
	}

	videoStorageRemotePath := currentVideoStorage.RemotePath
	if input.VideoStorageRemotePath != nil {
		videoStorageRemotePath = normalizeRemotePath(*input.VideoStorageRemotePath)
	}
	if videoStorageRemotePath == "" {
		videoStorageRemotePath = defaultVideoStorageRemotePath
	}

	videoStorageSyncInterval := currentVideoStorage.SyncIntervalSec
	if input.VideoStorageSyncInterval != nil {
		videoStorageSyncInterval = *input.VideoStorageSyncInterval
	}
	if videoStorageSyncInterval < minVideoStorageSyncIntervalSec || videoStorageSyncInterval > maxVideoStorageSyncIntervalSec {
		return AppSettings{}, fmt.Errorf(
			"%w: video_storage_sync_interval_seconds must be between %d and %d",
			ErrInvalid,
			minVideoStorageSyncIntervalSec,
			maxVideoStorageSyncIntervalSec,
		)
	}

	videoStorageMinFileAge := currentVideoStorage.MinFileAgeSec
	if input.VideoStorageMinFileAge != nil {
		videoStorageMinFileAge = *input.VideoStorageMinFileAge
	}
	if videoStorageMinFileAge < minVideoStorageMinFileAgeSec || videoStorageMinFileAge > maxVideoStorageMinFileAgeSec {
		return AppSettings{}, fmt.Errorf(
			"%w: video_storage_min_file_age_seconds must be between %d and %d",
			ErrInvalid,
			minVideoStorageMinFileAgeSec,
			maxVideoStorageMinFileAgeSec,
		)
	}

	if videoStorageEnabled {
		switch videoStorageProvider {
		case videoStorageProviderLocalServer:
			if strings.TrimSpace(videoStorageLocalPath) == "" {
				return AppSettings{}, fmt.Errorf("%w: video_storage_local_path is required for local_server", ErrInvalid)
			}
		default:
			if strings.TrimSpace(videoStorageRemoteName) == "" {
				return AppSettings{}, fmt.Errorf("%w: video_storage_remote_name is required for %s", ErrInvalid, videoStorageProvider)
			}
		}
	}

	if err := s.repo.Set(keyOpenAIEnabled, boolString(openaiEnabled)); err != nil {
		return AppSettings{}, err
	}
	if err := s.repo.Set(keyOpenAIModel, openaiModel); err != nil {
		return AppSettings{}, err
	}
	if strings.TrimSpace(openaiAPIKey) == "" {
		if err := s.repo.Delete(keyOpenAIAPIKey); err != nil {
			return AppSettings{}, err
		}
	} else {
		if err := s.repo.Set(keyOpenAIAPIKey, openaiAPIKey); err != nil {
			return AppSettings{}, err
		}
	}

	if err := s.repo.Set(keyVideoStorageEnabled, boolString(videoStorageEnabled)); err != nil {
		return AppSettings{}, err
	}
	if err := s.repo.Set(keyVideoStorageProvider, videoStorageProvider); err != nil {
		return AppSettings{}, err
	}
	if err := s.repo.Set(keyVideoStorageLocalPath, videoStorageLocalPath); err != nil {
		return AppSettings{}, err
	}
	if err := s.repo.Set(keyVideoStorageRemoteName, videoStorageRemoteName); err != nil {
		return AppSettings{}, err
	}
	if err := s.repo.Set(keyVideoStorageRemotePath, videoStorageRemotePath); err != nil {
		return AppSettings{}, err
	}
	if err := s.repo.Set(keyVideoStorageSyncInterval, intString(videoStorageSyncInterval)); err != nil {
		return AppSettings{}, err
	}
	if err := s.repo.Set(keyVideoStorageMinFileAge, intString(videoStorageMinFileAge)); err != nil {
		return AppSettings{}, err
	}

	return s.Get()
}

func boolString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func intString(v int) string {
	return strconv.Itoa(v)
}

func parseIntSetting(raw string, fallback int) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func isValidVideoStorageProvider(provider string) bool {
	_, ok := validVideoStorageProviders[provider]
	return ok
}

func normalizeVideoStorageProvider(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func normalizeRemotePath(raw string) string {
	return strings.Trim(strings.TrimSpace(raw), "/\\")
}
