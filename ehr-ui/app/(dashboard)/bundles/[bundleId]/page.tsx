"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import {
    Box,
    Card,
    CardContent,
    Typography,
    Divider,
    Chip,
    CircularProgress,
} from "@mui/material";
import {
    BundleResponse,
    BundleType,
    AllergyData,
    MedicationData,
    BloodGroupData,
    EmergencySummaryData,
} from "@/lib/bundle";
import {api} from "@/lib/api";
import Field from "@/components/Field";
import {formatDate} from "@/lib/util";

export default function BundleDetailPage() {
    const { bundleId } = useParams<{ bundleId: string }>();
    const [response, setData] = useState<BundleResponse | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        api
            .get(`/api/v1/bundles/${bundleId}/content`)
            .then((res) => setData(res.data))
            .finally(() => setLoading(false));
    }, [bundleId]);

    if (loading) {
        return (
            <Box sx={{ display: "flex", justifyContent: "center", mt: 6 }}>
                <CircularProgress />
            </Box>
        );
    }

    if (!response?.bundle) {
        return <Typography>Bundle not found</Typography>;
    }

    return (
        <Box sx={{ mx: "auto" }}>
            <Card sx={{mb: 3}}>
                <CardContent>
                    <Typography variant="h5" fontWeight={600}>
                        Patient Detail
                    </Typography>

                    <Box sx={{ mt: 1, mb: 2 }}>
                        {response.bundle.is_external ? (
                            <Chip label="External" color="warning" size="small" />
                        ) : (
                            <Chip label="Internal" color="primary" size="small" />
                        )}
                        <Typography variant="caption" sx={{ ml: 2 }}>
                            Created at: {formatDate(response.user?.created_at) || formatDate(response.bundle?.user?.created_at)}
                        </Typography>
                    </Box>

                    <Divider sx={{ mb: 2 }} />
                    <>
                        <Field label="Name" value={response.user?.name || response.bundle.user?.name} />
                        <Field label="Gender" value={"Male"} />
                    </>
                </CardContent>
            </Card>
            <Card>
                <CardContent>
                    <Typography variant="h5" fontWeight={600}>
                        Bundle Detail
                    </Typography>

                    <Box sx={{ mt: 1, mb: 2 }}>
                        <Chip label={response.bundle.type} color="primary" />
                        <Typography variant="caption" sx={{ ml: 2 }}>
                            Created at: {formatDate(response.bundle?.created_at) || ""}
                        </Typography>
                    </Box>

                    <Divider sx={{ mb: 2 }} />

                    {renderBundleContent(response.bundle.type, response.data)}
                </CardContent>
            </Card>
        </Box>
    );
}

function renderBundleContent(type: BundleType, content: any) {
    switch (type) {
        case "ALLERGY": {
            const d = content.data as AllergyData;
            return (
                <>
                    <Field label="Allergen" value={d.allergen} />
                    <Field label="Reaction" value={d.reaction} />
                    <Field label="Severity" value={d.severity} />
                    <Field label="Onset Date" value={d.onsetDate} />
                    <Field label="Notes" value={d.notes} />
                </>
            );
        }

        case "MEDICATION": {
            const d = content.data as MedicationData;
            return (
                <>
                    <Field label="Drug Name" value={d.drug_name} />
                    <Field label="Dose" value={d.dose} />
                    <Field label="Frequency" value={d.frequency} />
                    <Field label="Route" value={d.route} />
                    <Field label="Start Date" value={d.start_date} />
                    <Field label="End Date" value={d.end_date} />
                    <Field label="Notes" value={d.notes} />
                </>
            );
        }

        case "BLOOD_GROUP": {
            const d = content.data as BloodGroupData;
            return (
                <>
                    <Field label="ABO" value={d.abo} />
                    <Field label="Rh" value={d.rh} />
                    <Field label="Last Confirmed" value={d.last_confirmed} />
                </>
            );
        }

        case "EMERGENCY_SUMMARY": {
            const d = content.data as EmergencySummaryData;
            return (
                <>
                    <Field label="Problems" value={d.problems?.join(", ")} />
                    <Field label="Allergies" value={d.allergies?.join(", ")} />
                    <Field
                        label="Current Medications"
                        value={d.current_medications?.join(", ")}
                    />
                    <Field label="Notes" value={d.notes} />
                </>
            );
        }

        default:
            return <Typography>Unsupported bundle type</Typography>;
    }
}