"use client";

import {
    Box,
    Typography,
    Paper,
    Stack,
    Chip,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    CircularProgress,
} from "@mui/material";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import {formatDate} from "@/lib/util";

// Request detail
interface BreakGlassRequest {
    request_id: string;
    owner_hash: string;
    doctor_hash: string;
    request_from_hospital_id: string;
    is_external: boolean;
    justification: string;
    status: "REQUESTED" | "APPROVED" | "REJECTED" | "EXPIRED";
    expires_at: string;
    created_at: string;
}

// Bundle item
interface BreakGlassBundle {
    id: string;
    type: string;
    archetype_id?: string;
    hospital_id: string;
    is_external: boolean;
}

export default function BreakGlassRequestDetailPage() {
    const { requestid } = useParams<{ requestid: string }>();
    const router = useRouter();

    const [request, setRequest] = useState<any>(null);
    const [bundles, setBundles] = useState<any[]>([]);
    const [loading, setLoading] = useState(true);
    const [role, setRole] = useState<string>("");

    useEffect(() => {
        if (!requestid) return;
        const load = async () => {
            try {
                setLoading(true);
                // Load request details
                const reqRes = await api.get(`/healthchain/v1/consents/emergency/request/${requestid}`);
                setRequest(reqRes.data.result);

                // Load bundles in this request
                const bundlesRes = await api.get(`/healthchain/v1/consents/emergency/request/${requestid}/bundles`);
                setBundles(bundlesRes.data.result || []);
            } catch (err) {
                console.error("Failed to load break-glass request", err);
            } finally {
                setLoading(false);
            }
        };
        localStorage.getItem("role") && setRole(localStorage.getItem("role") || "");
        load();

    }, [requestid]);

    if (loading) {
        return (
            <Box p={4} textAlign="center">
                <CircularProgress />
            </Box>
        );
    }

    if (!request) {
        return (
            <Box p={4}>
                <Typography color="error">Request not found</Typography>
            </Box>
        );
    }

    return (
        <Box p={3}>
            {/* ===== Request Info ===== */}
            <Paper sx={{ p: 3, mb: 3 }}>
                <Typography variant="h6" fontWeight={600} mb={2}>
                    Break-Glass Request Detail
                </Typography>


                <Stack direction="row" spacing={2} mb={1}>
                    <Typography fontWeight={500}>Status:</Typography>
                    <Chip
                        label={request.status}
                        color={
                            request.status === "REQUESTED"
                                ? "warning"
                                : request.status === "APPROVED"
                                    ? "success"
                                    : request.status === "REJECTED"
                                        ? "error"
                                        : request.status === "EXPIRED"? "error" :"default"
                        }
                    />
                </Stack>

                <Typography>
                    <b>Justification:</b> {request.justification}
                </Typography>

                <Typography>
                    <b>From hospital:</b> {request.request_from_hospital_id}
                </Typography>

                <Typography>
                    <b>External request:</b>{" "}
                    {request.is_external ? "Yes" : "No"}
                </Typography>

                <Typography>
                    <b>Expires at:</b>{" "}
                    {formatDate(request.expires_at)}
                </Typography>

                <Typography>
                    <b>Created at:</b>{" "}
                    {formatDate(request.created_at)}
                </Typography>
            </Paper>

            {/* ===== Bundles ===== */}
            <Typography variant="h6" fontWeight={600} mb={1} color={"textSecondary"}>
                Bundles in this request
            </Typography>

            {role === "HOD" || request.status === "APPROVED" ?
            <TableContainer component={Paper}>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>Bundle ID</TableCell>
                            <TableCell>Type</TableCell>
                            <TableCell>Archetype</TableCell>
                            <TableCell>Hospital</TableCell>
                            <TableCell>External</TableCell>
                        </TableRow>
                    </TableHead>

                    <TableBody>
                        {bundles.length === 0 && (
                            <TableRow>
                                <TableCell colSpan={5} align="center">
                                    No bundles
                                </TableCell>
                            </TableRow>
                        )}

                        {bundles.map((b) => (
                            <TableRow
                                key={b.id}
                                hover
                                sx={{ cursor: "pointer" }}
                                onClick={() => router.push(`/bundles/${b.id}`)}
                            >
                                <TableCell>{b.id}</TableCell>
                                <TableCell>{b.type}</TableCell>
                                <TableCell>{b.archetype_id || "-"}</TableCell>
                                <TableCell>{b.hospital_id}</TableCell>
                                <TableCell>
                                    <Chip
                                        label={b.is_external ? "Yes" : "No"}
                                        color={b.is_external ? "warning" : "primary"}
                                        variant="outlined"
                                    />
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
                 : (<Typography color={"red"}>Bundles are only visible when the request is approved.</Typography>)
            }
        </Box>
    );
}