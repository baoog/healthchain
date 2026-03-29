"use client";

import { AuthProvider } from "@/context/AuthContext";
import Sidebar from "@/components/Sidebar";
import Topbar from "@/components/Topbar";
import { Box } from "@mui/material";
import {useEffect, useState} from "react";
import RouteGuard from "@/components/RouteGuard";

export default function DashboardLayout({
                                            children,
                                        }: {
    children: React.ReactNode;
}) {
    // const [mounted, setMounted] = useState(false);
    // useEffect(() => setMounted(true), []);
    // if (!mounted) return null;

    const [collapsed, setCollapsed] = useState(false);
    const [mobileOpen, setMobileOpen] = useState(false);
    const drawerWidth = collapsed ? 72 : 340;

    return (
        <AuthProvider>
            <Box sx={{ display: "flex", minHeight: "100vh" }}>
                <Sidebar
                    collapsed={collapsed}
                    mobileOpen={mobileOpen}
                    onCloseMobile={() => setMobileOpen(!mobileOpen)}
                />

                <Box sx={{ flexGrow: 1 }}>
                    <Topbar
                        collapsed={collapsed}
                        onMenuClick={() => {
                            if (window.innerWidth < 900) {
                                setMobileOpen(true);
                            } else {
                                setCollapsed((v) => !v);
                            }
                        }}
                    />
                    <Box
                        sx={{
                            pr: 0,
                            pl:3,
                            pt: 3,
                            ml: { md: `${drawerWidth}px` },
                            minHeight: "calc(100vh - 240px)",
                            backgroundColor: "#f5f7fb", // 👈 light dashboard background
                            height: "100vh",
                            overflowY: "auto"
                        }}
                    >
                        <RouteGuard>{children}</RouteGuard>
                    </Box>
                </Box>
            </Box>

        </AuthProvider>
    );
}