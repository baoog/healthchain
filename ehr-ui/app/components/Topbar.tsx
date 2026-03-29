"use client";

import React from "react";
import { AppBar, Toolbar, IconButton, Typography, Avatar, Box , useMediaQuery, useTheme} from "@mui/material";
import { Menu } from "@mui/icons-material";
import { useAuth } from "@/context/AuthContext";
import { SIDEBAR_EXPANDED, SIDEBAR_COLLAPSED } from "@/components/Sidebar";


export const TOPBAR_HEIGHT = 60;

interface TopbarProps {
    onMenuClick: () => void;
    collapsed: boolean;

}

export default function Topbar({ onMenuClick, collapsed }: TopbarProps) {
    const { role } = useAuth();
    const theme = useTheme();
    const isDesktop = useMediaQuery(theme.breakpoints.up("md"));

    const sidebarWidth = collapsed
        ? SIDEBAR_COLLAPSED
        : SIDEBAR_EXPANDED;

    return (
        <AppBar
            position="sticky"
            elevation={0}
            sx={{
                height: TOPBAR_HEIGHT,
                backgroundColor: "#ffffff",
                color: "#111",
                boxShadow: "0 2px 8px rgba(0,0,0,0.08)",
                left: isDesktop ? `${sidebarWidth}px` : 0,
                width: isDesktop
                    ? `calc(100% - ${sidebarWidth}px)`
                    : "100%",
                zIndex: (theme) => theme.zIndex.drawer,
            }}
        >
            <Toolbar sx={{ minHeight: TOPBAR_HEIGHT }}>
                <IconButton onClick={onMenuClick} aria-label="Toggle menu">
                    <Menu />
                </IconButton>
                <Typography sx={{ ml: 1, fontWeight: 500, color: "#555" }}>
                    Hide Menu
                </Typography>

                <Typography sx={{ flexGrow: 1, fontWeight: 600 }}>
                </Typography>

                <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                    <Avatar sx={{ width: 32, height: 32 }} />
                </Box>
            </Toolbar>
        </AppBar>
    );
}