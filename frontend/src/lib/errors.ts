import { ApiError, type ServerEnvelope } from "@/api/types";

/**
 * Map a parsed ServerEnvelope (or unstructured response body) to an ApiError.
 * If the body is unstructured (or missing), uses the HTTP status alone.
 */
export function mapServerError(body: unknown, status: number): ApiError {
  if (body && typeof body === "object" && "status" in body) {
    const envelope = body as ServerEnvelope<unknown>;
    if (envelope.status === "error") {
      const code = envelope.code ?? defaultCodeForStatus(status);
      const message = userMessage(code, envelope.error ?? "");
      return new ApiError({ status, code, message });
    }
  }
  const code = defaultCodeForStatus(status);
  return new ApiError({ status, code, message: userMessage(code, "") });
}

function defaultCodeForStatus(status: number): string {
  if (status === 401) return "UNAUTHORIZED";
  if (status === 403) return "FORBIDDEN";
  if (status === 404) return "NOT_FOUND";
  if (status === 409) return "CONFLICT";
  if (status === 410) return "GONE";
  if (status === 413) return "PAYLOAD_TOO_LARGE";
  if (status === 422) return "UNPROCESSABLE";
  if (status === 429) return "RATE_LIMITED";
  if (status >= 500) return "SERVER_ERROR";
  return "BAD_REQUEST";
}

const RU_BY_CODE: Record<string, string> = {
  UNAUTHORIZED: "Сессия истекла. Войдите снова.",
  FORBIDDEN: "Нет доступа.",
  NOT_FOUND: "Объект не найден.",
  CONFLICT: "Конфликт состояния. Обновите страницу.",
  GONE: "Ресурс удалён или истёк.",
  PAYLOAD_TOO_LARGE: "Файл слишком большой.",
  UNPROCESSABLE: "Невозможно выполнить операцию.",
  RATE_LIMITED: "Слишком много запросов. Попробуйте позже.",
  SERVER_ERROR: "Произошла ошибка на сервере. Попробуйте ещё раз.",
  BAD_REQUEST: "Некорректный запрос.",

  INVALID_TOKEN: "Сессия истекла. Войдите снова.",
  NODE_NOT_FOUND: "Объект не найден.",
  INVALID_NAME: "Некорректное имя.",
  NAME_CONFLICT: "Файл или папка с таким именем уже существует.",
  MOVE_INTO_DESCENDANT: "Нельзя переместить папку внутрь самой себя.",
  MOVE_INTO_SELF: "Нельзя переместить объект сам в себя.",
  MOVE_ACROSS_OWNERS: "Нельзя переместить объект между владельцами.",
  ROOT_IMMUTABLE: "Корневую папку нельзя изменить.",
  FILE_TOO_LARGE: "Файл слишком большой (лимит 5 ГиБ).",
  FILE_NOT_PENDING: "Загрузка уже завершена или отменена.",
  FILE_NOT_ACTIVE: "Файл ещё не загружен.",
  CHECKSUM_MISMATCH: "Контрольная сумма не сошлась — повторите загрузку.",
  NODE_KIND_MISMATCH: "Несовместимый тип объекта.",
  NODE_ALREADY_DELETED: "Объект уже удалён.",

  INVALID_SHARE_TOKEN: "Ссылка недействительна.",
  SHARE_NOT_FOUND: "Ссылка не найдена.",
  SHARE_EXPIRED: "Ссылка истекла.",
  SHARE_REVOKED: "Ссылка отозвана.",

  COMMAND_NOT_FOUND: "Команда не найдена.",
  COMMAND_EXPIRED: "План устарел. Сформируйте заново.",
  COMMAND_FORBIDDEN: "Команда принадлежит другому пользователю.",
  COMMAND_ALREADY_EXECUTED: "Команда уже выполнена.",
  COMMAND_ALREADY_CANCELLED: "Команда уже отменена.",
  COMMAND_NOT_AWAITING: "Команду нельзя выполнить в текущем статусе.",
  LLM_UNAVAILABLE: "ИИ временно недоступен. Попробуйте позже.",
  LLM_INVALID_RESPONSE: "ИИ вернул некорректный ответ. Попробуйте переформулировать.",
  STORAGE_UNAVAILABLE: "Хранилище временно недоступно.",

  USER_ALREADY_EXISTS: "Пользователь с таким email уже зарегистрирован.",
  INVALID_CREDENTIALS: "Неверный email или пароль.",
  USER_INACTIVE: "Аккаунт деактивирован.",
};

export function userMessage(code: string, fallback: string): string {
  return RU_BY_CODE[code] ?? fallback ?? "Произошла ошибка. Попробуйте ещё раз.";
}

/** Type-guard for downstream code that wants to react to specific codes. */
export function isApiErrorWithCode(err: unknown, ...codes: string[]): err is ApiError {
  return err instanceof ApiError && codes.includes(err.code);
}
