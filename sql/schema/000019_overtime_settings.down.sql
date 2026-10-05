DELETE FROM master_settings WHERE key IN (
    'overtime_threshold_hours_non_spd',
    'overtime_spd_enabled',
    'overtime_spd_same_as_non_spd',
    'overtime_threshold_hours_spd'
);
