"use client";

import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/context/AuthContext";
import { ROLE_ALLOWED_PATHS, ROLE_HOME } from "@/config/roleAccess";
import { useEffect } from "react";

export default function RouteGuard({ children }: { children: React.ReactNode }) {
    const { role } = useAuth();
    const pathname = usePathname();
    const router = useRouter();

    useEffect(() => {
        if (!role) return;

        const allowed = ROLE_ALLOWED_PATHS[role].some(
            (p) => pathname === p || pathname.startsWith(p + "/")
        );

        if (!allowed) {
            router.replace(ROLE_HOME[role]);
        }
    }, [role, pathname]);

    return <>{children}</>;
}