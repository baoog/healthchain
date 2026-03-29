"use client";

import {
    Box, Table, TableBody, TableCell, TableHead,
    TableRow, TableContainer, Paper, Chip, Typography, Button, Stack
} from "@mui/material";
import { useRouter } from "next/navigation";
import {useEffect, useState} from "react";
import {api} from "@/lib/api";
import AddIcon from "@mui/icons-material/Add";
import {ellipsisCell, formatDate} from "@/lib/util";

export default function BreakGlassListPage() {
    const router = useRouter();
    const [requests, setRequests] = useState<Array<{
        request_id: string;
        owner_hash: string;
        bundle_count: number;
        allowed_types: string[];
        status: string;
        created_at: string;
    }>>([]);

    useEffect(() => {
        if(!requests) return;
        loadRequests();
    }, []);

    const loadRequests = async () => {
        try {
            const res = await api.get("/healthchain/v1/consents/emergency/requests");
            setRequests(Array.isArray(res.data.result.requests) ? res.data.result.requests : []);
        } catch (err) {
            console.error("Failed to load break-glass requests", err);
            setRequests([]);
        }
    }

    return (
        <Box p={3}>
            <Stack
                direction="row"
                justifyContent="space-between"
                alignItems="center"
                mb={2}
            >
                <Typography variant="h5" fontWeight={600} mb={2} style={{"color":"black"}}>
                    Break-Glass Requests
                </Typography>
                <Button
                    variant="contained"
                    startIcon={<AddIcon />}
                    onClick={() => router.push("/doctor/break-glass/new")}
                >
                    New Break-glass request
                </Button>
            </Stack>

            <TableContainer component={Paper}>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>Request ID</TableCell>
                            <TableCell>Patient</TableCell>
                            <TableCell>Patient Name</TableCell>
                            <TableCell>External</TableCell>
                            <TableCell>Bundles</TableCell>
                            <TableCell>Status</TableCell>
                            <TableCell>Created At</TableCell>
                        </TableRow>
                    </TableHead>

                    <TableBody>
                        {requests.map(r => (
                            <TableRow
                                key={r.request_id}
                                hover
                                sx={{ cursor: "pointer" }}
                                onClick={() =>
                                    router.push(`/break-glass/${r.request_id}`)
                                }
                            >
                                <TableCell sx={ellipsisCell}>{r.request_id.slice(0, 8)}…</TableCell>
                                <TableCell sx={ellipsisCell}>{r.owner_hash}</TableCell>
                                <TableCell>{r.owner?.name || "Unknown"}</TableCell>
                                <TableCell>
                                    <Chip
                                        label={r.is_external ? "Yes" : "No"}
                                        color={r.is_external ? "warning" : "primary"}
                                        variant="outlined"
                                    />
                                </TableCell>
                                <TableCell>{r.allowed_types.join(", ")}</TableCell>
                                <TableCell>
                                    <Chip
                                    label={r.status}
                                    color={
                                    r.status === "REQUESTED"
                                        ? "warning"
                                        : r.status === "APPROVED"
                                            ? "success"
                                            : "error"
                                }
                                    />
                                </TableCell>
                                <TableCell>{formatDate(r.created_at)}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
        </Box>
    );
}