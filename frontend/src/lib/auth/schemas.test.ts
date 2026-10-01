import { it, expect } from "vitest";
import { userSchema, privateErrorSchema } from "./schemas";
const user = { id: "10000000-0000-4000-8000-000000000001", username: "test_user", roles: ["learner"], mustChangePassword: false };
it("TestPrivateSchemas", () => {
    expect(userSchema.safeParse(user).success).toBe(true);
    for (const bad of [{ ...user, password_phc: "not public" }, { ...user, roles: ["admin"] }, { ...user, roles: ["learner", "learner"] }, { ...user, roles: ["admin", "learner"] }, { ...user, id: "bad" }])
        expect(userSchema.safeParse(bad).success).toBe(false);
    expect(privateErrorSchema.safeParse({ error: { code: "INVALID_COOKIE", message: "Invalid sign-in cookie.", requestId: "a".repeat(32) } }).success).toBe(true);
    expect(privateErrorSchema.safeParse({ error: { code: "INTERNAL_ERROR", message: "hidden", requestId: "a".repeat(32) } }).success).toBe(false);
});
