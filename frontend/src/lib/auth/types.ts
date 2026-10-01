import type { components } from "../api/generated";
export type User = components["schemas"]["User"];
export type Role = components["schemas"]["Role"];
export type AuthContext = components["schemas"]["AuthContext"];
export type UserPage = components["schemas"]["UserPage"];
export type UserQuery = components["schemas"]["UserQuery"];
export type PrivateErrorCode = components["schemas"]["PrivateErrorCode"];
export type AuthResult<T> = {
    ok: true;
    data: T;
} | {
    ok: false;
    status: number;
    code: PrivateErrorCode;
    message: string;
    retryAfter?: number;
};
export type PrivateRoute = {
    kind: "register" | "login" | "logout" | "logout-all" | "password" | "reauth";
} | {
    kind: "roles" | "reset";
    userId: string;
} | {
    kind: "users";
    query: UserQuery;
};
export type PrivateEndpoint = "session" | "context" | PrivateRoute["kind"];
