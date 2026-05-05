package usecase

import (
	"log/slog"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/repository"
)

// Compile-time check: AIService implements port.AIUseCase
var _ port.AIUseCase = (*AIService)(nil)

// AIService implements the AIUseCase interface.
// Coordinates the LLM client, storage-service client, ai_commands repository
// and the supporting helpers (prompt builder / plan validator / parser).
type AIService struct {
	cmdRepo       repository.AiCommandRepository
	txManager     repository.TransactionManager
	storage       port.StorageClient
	llm           port.LLMClient
	ids           port.IDGenerator
	policy        *authorizationPolicy
	prompt        *promptBuilder
	validator     *planValidator
	parser        *llmResponseParser
	maxInputChars int
	planTTL       time.Duration
	maxRetries    int
	llmModel      string
	logger        *slog.Logger
	now           func() time.Time
}

// AIServiceConfig collects all inputs needed to construct AIService.
// (Avoids a 13-arg constructor.)
type AIServiceConfig struct {
	CmdRepo       repository.AiCommandRepository
	TxManager     repository.TransactionManager
	Storage       port.StorageClient
	LLM           port.LLMClient
	IDs           port.IDGenerator
	MaxInputChars int
	PlanTTL       time.Duration
	MaxRetries    int
	LLMModel      string
	Logger        *slog.Logger
	// Now is optional; defaults to time.Now. Useful for deterministic tests.
	Now func() time.Time
}

// NewAIService creates a new AIService.
func NewAIService(cfg AIServiceConfig) *AIService {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &AIService{
		cmdRepo:       cfg.CmdRepo,
		txManager:     cfg.TxManager,
		storage:       cfg.Storage,
		llm:           cfg.LLM,
		ids:           cfg.IDs,
		policy:        newAuthorizationPolicy(),
		prompt:        newPromptBuilder(),
		validator:     newPlanValidator(),
		parser:        newLLMResponseParser(),
		maxInputChars: cfg.MaxInputChars,
		planTTL:       cfg.PlanTTL,
		maxRetries:    cfg.MaxRetries,
		llmModel:      cfg.LLMModel,
		logger:        cfg.Logger,
		now:           now,
	}
}
