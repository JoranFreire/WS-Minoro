import { describe, it, expect, beforeEach } from "vitest";
import { isAuthenticated, clearAuthState } from "./auth";

function setCookie(value: string) {
  document.cookie = value;
}

function clearAllCookies() {
  document.cookie.split(";").forEach((c) => {
    const name = c.split("=")[0].trim();
    if (name) document.cookie = `${name}=; path=/; max-age=0`;
  });
}

describe("isAuthenticated", () => {
  beforeEach(() => {
    clearAllCookies();
  });

  it("returns false when no auth_state cookie is present", () => {
    expect(isAuthenticated()).toBe(false);
  });

  it("returns true when the auth_state cookie is present", () => {
    setCookie("auth_state=1");
    expect(isAuthenticated()).toBe(true);
  });

  it("is not fooled by a cookie whose name merely contains auth_state", () => {
    setCookie("not_auth_state=1");
    expect(isAuthenticated()).toBe(false);
  });

  it("ignores unrelated cookies alongside the real one", () => {
    setCookie("some_other=value");
    setCookie("auth_state=1");
    expect(isAuthenticated()).toBe(true);
  });
});

describe("clearAuthState", () => {
  beforeEach(() => {
    clearAllCookies();
  });

  it("removes the auth_state cookie", () => {
    setCookie("auth_state=1");
    expect(isAuthenticated()).toBe(true);

    clearAuthState();

    expect(isAuthenticated()).toBe(false);
  });
});
