export function formatDate(iso: string) {
    if(!iso) return "";
    return new Date(iso).toISOString().replace("T", " ").slice(0, 19);
}

export const ellipsisCell = {
    maxWidth: 180,          // adjust per column
    whiteSpace: "nowrap",
    overflow: "hidden",
    textOverflow: "ellipsis",
};