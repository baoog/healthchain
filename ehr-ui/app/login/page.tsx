"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
// @ts-ignore
import { api } from "@/lib/api";

const COUNTRY_OPTIONS = [
    { code: "VN", label: "Vietnam" },
    { code: "US", label: "United States" },
    { code: "KR", label: "Korea" },
    { code: "JP", label: "Japan" },
];

export default function LoginPage() {
    const router = useRouter();

    const [national_id_hash, setNationalIdHash] = useState("");
    const [countryCode, setCountryCode] = useState("VN");
    const [identity, setIdentity] = useState("");
    const [password, setPassword] = useState("");
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");

    const sha256Hex = async (value: string): Promise<string> => {
        const encoded = new TextEncoder().encode(value);
        const hashBuffer = await crypto.subtle.digest("SHA-256", encoded);
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        return hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
    };

    const handleLogin = async (e: React.FormEvent) => {
        e.preventDefault();
        setError("");
        setLoading(true);

        try {
            const usernameSource = `${countryCode}:${identity}`;
            const national_id_hash = await sha256Hex(usernameSource);

            const res = await api.post("/api/v1/user/login", {
                national_id_hash,
                password,
            });

            const { token } = res.data;

            // store token (MVP)
            localStorage.setItem("access_token", token);
            // 2. Call /me using token
            const meRes = await api.get("/api/v1/user/me");
            const user = meRes.data;
            const role = user.info.role;


            // 3. Store user info (MVP)
            localStorage.setItem("user", JSON.stringify(user));
            localStorage.setItem("role", role);

            // 4. Redirect by role
            if (role === "PATIENT") router.push("/patient");
            else if (role === "DOCTOR") router.push("/doctor");
            else if (role  === "HOD") router.push("/hod");
            else router.push("/");

        } catch (err: any) {
            setError(
                err?.response?.data?.message ||
                "Login failed"
            );
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen flex items-center justify-center bg-gray-50">
            <form
                onSubmit={handleLogin}
                className="w-full max-w-sm bg-white p-6 rounded shadow" style={{"color": "black"}}
            >
                <h1 className="text-xl font-semibold mb-4 text-center" color={"black"}>
                    EHR Login
                </h1>

                {error && (
                    <div className="mb-3 text-sm text-red-600">
                        {error}
                    </div>
                )}

                <div className="mb-3">
                    <label className="block text-sm mb-1" style={{"color": "black"}}>Country</label>
                    <select
                        value={countryCode}
                        onChange={(e) => setCountryCode(e.target.value)}
                        className="w-full border px-3 py-2 rounded bg-white"
                        required
                    >
                        {COUNTRY_OPTIONS.map((option) => (
                            <option key={option.code} value={option.code}>
                                {option.label}
                            </option>
                        ))}
                    </select>
                </div>

                <div className="mb-3">
                    <label className="block text-sm mb-1" style={{"color": "black"}}>Identity Number</label> <input
                    type="text"
                    placeholder={"Enter identity number"}
                    value={identity}
                    onChange={(e) => setIdentity(e.target.value)}
                        className="w-full border px-3 py-2 rounded"
                        required
                    />
                </div>

                <div className="mb-4">
                    <label className="block text-sm mb-1" style={{"color": "black"}}>Password</label>
                    <input
                        type="password"
                        placeholder={"Password"}
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        className="w-full border px-3 py-2 rounded"
                        style={{"color": "black"}}
                        required
                    />
                </div>

                <button
                    type="submit"
                    disabled={loading}
                    className="w-full bg-blue-600 text-white py-2 rounded hover:bg-blue-700 disabled:opacity-50"
                >
                    {loading ? "Signing in..." : "Login"}
                </button>
            </form>
        </div>
    );
}