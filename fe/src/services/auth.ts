import http from '@/lib/http';
import type {
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RefreshTokenRequest,
  RefreshTokenResponse,
  LogoutResponse,
} from '@/types';

export const authService = {
  async login(data: LoginRequest): Promise<LoginResponse> {
    const response = await http.post<LoginResponse>('/v1/auth/login', data);
    return response.data;
  },

  async register(data: RegisterRequest): Promise<LoginResponse> {
    const response = await http.post<LoginResponse>('/v1/auth/register', data);
    return response.data;
  },

  async refreshToken(data: RefreshTokenRequest): Promise<RefreshTokenResponse> {
    const response = await http.post<RefreshTokenResponse>('/v1/auth/refresh', data);
    return response.data;
  },

  async logout(): Promise<LogoutResponse> {
    const response = await http.post<LogoutResponse>('/v1/auth/logout', {});
    return response.data;
  },
};
