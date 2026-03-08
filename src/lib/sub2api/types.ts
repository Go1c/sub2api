export interface Sub2ApiUser {
  id: number;
  username: string;
  email: string;
  status: string; // "active", "banned", etc.
  balance: number;
  notes?: string;
}

export interface Sub2ApiRedeemCode {
  id: number;
  code: string;
  type: string;
  value: number;
  status: string;
  used_by: number;
  used_at: string;
}

export interface Sub2ApiResponse<T> {
  code: number;
  data?: T;
  message?: string;
}
