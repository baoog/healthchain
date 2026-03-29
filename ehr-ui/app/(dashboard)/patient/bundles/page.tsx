"use client";

import {
    Box,
    Button,
    IconButton,
    Paper,
    Stack,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Typography,
    Chip,
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import {Bundle} from "@/lib/bundle";
import {formatDate} from "@/lib/util";

export default function MyBundlesPage() {
    const router = useRouter();
    const [bundles, setBundles] = useState<Bundle[]>([]);
    const [loading, setLoading] = useState(false);

    const loadBundles = async () => {
        setLoading(true);
        try {
            const res = await api.get("/healthchain/v1/fabric/bundles");

            // ✅ Defensive: backend may wrap data
            const list = Array.isArray(res.data.result)
                ? res.data.result
                :  [];

            setBundles(list);
        } catch (err) {
            console.error("Failed to load bundles", err);
            setBundles([]);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadBundles();
    }, []);

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
                    My Bundles
                </Typography>

                <Button
                    variant="contained"
                    startIcon={<AddIcon />}
                    onClick={() => router.push("/patient/bundles/new")}
                >
                    Upload Bundle
                </Button>
            </Stack>

            {/* Table */}
            <TableContainer component={Paper}>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>Bundle ID</TableCell>
                            <TableCell>Type</TableCell>
                            <TableCell>Archetype</TableCell>
                            <TableCell>External</TableCell>
                            <TableCell>Created At</TableCell>
                        </TableRow>
                    </TableHead>

                    <TableBody>
                        {!loading && bundles.length === 0 && (
                            <TableRow>
                                <TableCell colSpan={4} align="center">
                                    No bundles found
                                </TableCell>
                            </TableRow>
                        )}

                        {bundles.map((b) => (
                            <TableRow
                                key={b.id}
                                hover
                                sx={{ cursor: "pointer" }}
                                onClick={() =>
                                    router.push(`/bundles/${b.id}`)
                                }
                            >
                                <TableCell>{b.id}</TableCell>
                                <TableCell>{b.type}</TableCell>
                                <TableCell>
                                    {b.archetype_id || "-"}
                                </TableCell>
                                <TableCell>{b.is_external ? <Chip label="Yes" color="warning" variant="outlined" /> : <Chip label="No" color="primary" variant="outlined" />}</TableCell>
                                <TableCell>
                                    {formatDate(b.created_at)}
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
        </Box>
    );
}