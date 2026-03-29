"use client";

import { createContext, useContext, useEffect, useState } from "react";
import {UserToken} from "@/lib/user";

export type UserRole = "PATIENT" | "DOCTOR" | "HOD";

interface AuthContextType {
    role: UserRole | null;
    setRole: (r: UserRole | null) => void;
    userToken: UserToken | null;
    setUserToken: (u: UserToken|null) => void;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [role, setRole] = useState<UserRole | null>(null);
    const [userToken, setUserToken] = useState<UserToken | null>(null);

    useEffect(() => {
        const stored = localStorage.getItem("role") as UserRole | null;
        if (stored) setRole(stored);

        const userTkn = localStorage.getItem("user") as UserToken | null;
        if(userTkn) setUserToken(userTkn)
    }, []);

    return (
        <AuthContext.Provider value={{ role, setRole ,userToken , setUserToken}}>
            {children}
        </AuthContext.Provider>
    );
}

export function useAuth() {
    const ctx = useContext(AuthContext);
    if (!ctx) {
        throw new Error("useAuth must be used inside AuthProvider");
    }
    return ctx;
}