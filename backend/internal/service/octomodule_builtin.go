package service

import (
	"context"
	"encoding/json"
	"path/filepath"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/model"
	"octomanger/backend/internal/octomodule"
	"octomanger/backend/internal/repository"
)

type builtinOctoModule struct {
	TypeKey      string
	Category     string
	EntryPath    string
	Source       string
	RunFilter    repository.JobRunListFilter
	ScriptConfig json.RawMessage
}

func builtinOctoModules(moduleDir string) []builtinOctoModule {
	baseDir := trim(moduleDir)
	if baseDir == "" {
		return nil
	}
	entryPath := filepath.Join(baseDir, "_email_batch_outlook", "main.py")
	return []builtinOctoModule{
		{
			TypeKey:      "_email_batch_outlook",
			Category:     "system",
			EntryPath:    entryPath,
			Source:       "builtin",
			RunFilter:    repository.JobRunListFilter{TypeKey: "system", ActionKey: "batch_email_register"},
			ScriptConfig: json.RawMessage("null"),
		},
	}
}

func (s *octoModuleService) getBuiltinModule(typeKey string) (*builtinOctoModule, bool) {
	for _, item := range builtinOctoModules(s.moduleDir) {
		if item.TypeKey == trim(typeKey) {
			copyItem := item
			return &copyItem, true
		}
	}
	return nil, false
}

func (s *octoModuleService) buildBuiltinInfo(item builtinOctoModule) dto.OctoModuleInfoResponse {
	return dto.OctoModuleInfoResponse{
		TypeKey:      item.TypeKey,
		Category:     item.Category,
		ScriptPath:   item.EntryPath,
		ModuleDir:    filepath.Dir(item.EntryPath),
		EntryFile:    filepath.Base(item.EntryPath),
		Source:       item.Source,
		Exists:       octomodule.FileExists(item.EntryPath),
		ScriptConfig: item.ScriptConfig,
	}
}

func (s *octoModuleService) resolveModuleInfo(ctx context.Context, typeKey string) (*dto.OctoModuleInfoResponse, *builtinOctoModule, *model.AccountType, error) {
	if builtin, ok := s.getBuiltinModule(typeKey); ok {
		info := s.buildBuiltinInfo(*builtin)
		return &info, builtin, nil, nil
	}

	item, err := s.getGenericAccountType(ctx, typeKey)
	if err != nil {
		return nil, nil, nil, err
	}
	info := s.buildInfo(item)
	return &info, nil, item, nil
}
