import {Box, Typography} from "@mui/material";

export default function Field({
                   label,
                   value,
               }: {
    label: string;
    value?: string;
}) {
    if (!value) return null;

    return (
        <Box sx={{ mb: 1.5 }}>
            <Typography variant="caption" color="text.secondary">
                {label}
            </Typography>
            <Typography>{value}</Typography>
        </Box>
    );
}