"use client";

import {
    Box,
    Button,
    IconButton,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Typography,
    Paper,
    Stack, Chip,
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import CloseIcon from "@mui/icons-material/Close";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {api} from "@/lib/api";
import {Bundle} from "@/lib/bundle";
import {User} from "@/lib/user"; // your axios wrapper

interface ListConsentResponse {
    consent: Consent;
    bundle: Bundle;
    grantee?: User | {name: ""; national_id_hash: ""};
};

interface Consent {
    bundle_id: string;
    grantee_hash: string;
    status: string;
    created_at:string;
    updated_at: string;
    expires_at: string;
}

function formatDate(iso: string) {
    return new Date(iso).toISOString().replace("T", " ").slice(0, 19);
}

export default function OwnerConsentsPage() {
    const router = useRouter();
    const [consents, setConsents] = useState<ListConsentResponse[]>([]);
    const [loading, setLoading] = useState(false);
    const [now, setNow] = useState<string>("");

    const loadConsents = async () => {
        setLoading(true);
        try {
            const res = await api.get("/healthchain/v1/fabric/consents");
            setConsents(res?.data?.result || []);
        } catch (err) {
            console.error("Failed to load consents", err);
            setConsents([]);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadConsents();
    }, []);

    const revokeConsent = async (bundleID: string, doctorID: string) => {
        if (!confirm("Are you sure you want to revoke this consent?")) return;

        try {
            await api.post("/healthchain/v1/consents/revoke", {
                bundle_id: bundleID,
                doctor_id: doctorID,
            });
            await loadConsents();
        } catch (err) {
            alert("Failed to revoke consent");
            console.error(err);
        }
    };

    return (
        <Box p={3}>
            {/* Header */}
            <Stack
                direction="row"
                justifyContent="space-between"
                alignItems="center"
                mb={2}
            >
                <Typography variant="h5" fontWeight={600} color={"black"}>
                    My Consents
                </Typography>

                <Button
                    variant="contained"
                    startIcon={<AddIcon />}
                    onClick={() => router.push("/patient/consents/new")}
                >
                    Grant Consent
                </Button>
            </Stack>

            {/* Table */}
            <TableContainer component={Paper} style={{overflowX: "auto"}}>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>Bundle ID</TableCell>
                            <TableCell>Bundle Type</TableCell>
                            <TableCell>Bundle Archetype</TableCell>
                            <TableCell>Doctor</TableCell>
                            <TableCell>Status</TableCell>
                            <TableCell>Created At</TableCell>
                            <TableCell align="center">Action</TableCell>
                        </TableRow>
                    </TableHead>

                    <TableBody>
                        {!loading && consents.length === 0 && (
                            <TableRow>
                                <TableCell colSpan={6} align="center">
                                    No consents found
                                </TableCell>
                            </TableRow>
                        )}

                        {consents.map((c,index) => (
                            <TableRow key={index}>
                                <TableCell>
                                    {c.bundle.id}
                                </TableCell>
                                <TableCell>
                                    {c.bundle.type}
                                </TableCell>
                                <TableCell>
                                    {c.bundle.archetype_id}
                                </TableCell>
                                <TableCell>
                                    {c.grantee?.name || "Joe Doctor"}
                                </TableCell>
                                <TableCell>
                                    {c.consent.status === "REVOKED" ? <Chip label="REVOKED" color="error" variant="outlined" /> : <Chip label="GRANTED" color="primary" variant="outlined" />}
                                </TableCell>
                                <TableCell>
                                    {formatDate(c.consent.created_at)}
                                </TableCell>
                                <TableCell align="center">
                                    {c.consent.status === "GRANTED" && (
                                        <IconButton
                                            color="error"
                                            onClick={() => revokeConsent(c.consent.bundle_id, c.grantee?.national_id_hash)}
                                        >
                                            <CloseIcon />
                                        </IconButton>
                                    )}
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
        </Box>
    );
}