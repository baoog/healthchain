import axios from "axios";

export const api = axios.create({
    baseURL: process.env.NEXT_PUBLIC_ORCH_URL || "http://localhost:8080",
    timeout: 1500000,
});

api.interceptors.request.use((config) => {
    // simple for MVP
    const token = typeof window !== "undefined" ? localStorage.getItem("access_token") : null;
    if (token) config.headers.Authorization = `Bearer ${token}`;
    return config;
});