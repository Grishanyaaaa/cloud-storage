import { apiPublicFetch, readTokenPair, readUserId } from "./client";
import type { AuthTokenPair, RegisterResult } from "./types";
import { ApiError } from "./types";

export async function register(input: { email: string; password: string }): Promise<RegisterResult> {
  const data = await apiPublicFetch<unknown>("/auth/v1/register", {
    method: "POST",
    body: input,
  });
  const userId = readUserId(data);
  if (!userId) {
    throw new ApiError({
      status: 502,
      code: "SERVER_ERROR",
      message: "Сервер вернул некорректный ответ при регистрации.",
    });
  }
  return { user_id: userId };
}

export async function login(input: { email: string; password: string }): Promise<AuthTokenPair> {
  const data = await apiPublicFetch<unknown>("/auth/v1/login", {
    method: "POST",
    body: input,
  });
  const pair = readTokenPair(data);
  if (!pair) {
    throw new ApiError({
      status: 502,
      code: "SERVER_ERROR",
      message: "Сервер вернул некорректный ответ при входе.",
    });
  }
  return pair;
}

export async function refresh(input: { refresh_token: string }): Promise<AuthTokenPair> {
  const data = await apiPublicFetch<unknown>("/auth/v1/refresh", {
    method: "POST",
    body: input,
  });
  const pair = readTokenPair(data);
  if (!pair) {
    throw new ApiError({
      status: 502,
      code: "SERVER_ERROR",
      message: "Сервер вернул некорректный ответ при обновлении токена.",
    });
  }
  return pair;
}

export async function logout(input: { refresh_token: string }): Promise<void> {
  await apiPublicFetch<undefined>("/auth/v1/logout", {
    method: "POST",
    body: input,
  });
}
