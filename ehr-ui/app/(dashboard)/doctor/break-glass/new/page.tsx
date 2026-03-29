"use client";

import {
    Box,
    Button,
    TextField,
    Typography,
    Stack,
    MenuItem,
    Checkbox,
    FormControlLabel,
    FormGroup,
    RadioGroup,
    Radio,
    Paper,
} from "@mui/material";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import {sha256} from "@/lib/hash";

const BUNDLE_TYPES = [
    { label: "Allergy", value: "ALLERGY" },
    { label: "Medication", value: "MEDICATION" },
    { label: "Blood Group", value: "BLOOD_GROUP" },
    { label: "Emergency Summary", value: "EMERGENCY_SUMMARY" },
];

export default function CreateBreakGlassPage() {
    const router = useRouter();

    const [countryCode, setCountryCode] = useState("VN");
    const [citizenId, setCitizenId] = useState("");
    const [justification, setJustification] = useState("");
    const [duration, setDuration] = useState("24h");
    const [scopes, setScopes] = useState<string[]>([]);
    const [loading, setLoading] = useState(false);

    const toggleScope = (v: string) => {
        setScopes((prev) =>
            prev.includes(v) ? prev.filter((x) => x !== v) : [...prev, v]
        );
    };

    const submit = async () => {
        if (!citizenId || !justification || scopes.length === 0) {
            alert("Please fill all required fields");
            return;
        }

        const ownerHash = await sha256(`${countryCode}:${citizenId}`);


        setLoading(true);
        try {
            await api.post("/healthchain/v1/consents/emergency/request", {
                patient_id: ownerHash,
                justification,
                expires_in_minutes: duration,
                types: scopes,
            });

            router.push("/doctor/break-glass");
        } catch (err) {
            console.error(err);
            alert("Failed to submit break-glass request");
        } finally {
            setLoading(false);
        }
    };

    return (
        <Box>
            <Paper sx={{ p: 4 }}>
                <Typography variant="h5" fontWeight={600} mb={3}>
                    Break-Glass Request
                </Typography>

                {/* Patient info */}
                <Stack direction="row" spacing={2} mb={3}>
                    <TextField
                        select
                        label="Country Code"
                        value={countryCode}
                        onChange={(e) => setCountryCode(e.target.value)}
                        sx={{ width: 140 }}
                    >
                        <MenuItem value="VN">VN</MenuItem>
                        <MenuItem value="KR">KR</MenuItem>
                        <MenuItem value="US">US</MenuItem>
                    </TextField>

                    <TextField
                        fullWidth
                        label="Citizen ID"
                        value={citizenId}
                        onChange={(e) => setCitizenId(e.target.value)}
                        required
                    />
                </Stack>

                {/* Scope */}
                <Typography fontWeight={600} mb={1}>
                    Emergency Scope
                </Typography>
                <FormGroup row sx={{ mb: 3 }}>
                    {BUNDLE_TYPES.map((b) => (
                        <FormControlLabel
                            key={b.value}
                            control={
                                <Checkbox
                                    checked={scopes.includes(b.value)}
                                    onChange={() => toggleScope(b.value)}
                                />
                            }
                            label={b.label}
                        />
                    ))}
                </FormGroup>

                {/* Duration */}
                <Typography fontWeight={600} mb={1}>
                    Access Duration
                </Typography>
                <RadioGroup
                    row
                    value={duration}
                    onChange={(e) => setDuration(e.target.value)}
                    sx={{ mb: 3 }}
                >
                    <FormControlLabel value="5m" control={<Radio />} label="24 hours" />
                    <FormControlLabel value="10m" control={<Radio />} label="48 hours" />
                    <FormControlLabel value="120m" control={<Radio />} label="72 hours" />
                </RadioGroup>

                {/* Justification */}
                <TextField
                    label="Justification"
                    multiline
                    rows={4}
                    fullWidth
                    required
                    value={justification}
                    onChange={(e) => setJustification(e.target.value)}
                    sx={{ mb: 4 }}
                />

                <Button
                    variant="contained"
                    size="large"
                    fullWidth
                    disabled={loading}
                    onClick={submit}
                >
                    Submit Break-Glass Request
                </Button>
            </Paper>
        </Box>
    );
}