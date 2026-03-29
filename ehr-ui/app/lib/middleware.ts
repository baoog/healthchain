import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import { ROLE_ALLOWED_PATHS, ROLE_HOME } from "@/config/roleAccess";

export function middleware(req: NextRequest) {
    const token = req.cookies.get("access_token")?.value;
    const role = req.cookies.get("role")?.value as keyof typeof ROLE_ALLOWED_PATHS;

    const pathname = req.nextUrl.pathname;

    // Not logged in
    if (!token || !role) {
        return NextResponse.redirect(new URL("/login", req.url));
    }

    const allowedPaths = ROLE_ALLOWED_PATHS[role];
    if (!allowedPaths) {
        return NextResponse.redirect(new URL("/login", req.url));
    }

    const isAllowed = allowedPaths.some((path) =>
        pathname === path || pathname.startsWith(path + "/")
    );

    if (!isAllowed) {
        return NextResponse.redirect(new URL(ROLE_HOME[role], req.url));
    }

    return NextResponse.next();
}

export const config = {
    matcher: [
        "/dashboard/:path*",
        "/doctor/:path*",
        "/hod/:path*",
    ],
};