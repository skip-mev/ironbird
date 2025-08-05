-- Revert TestnetDuration from string back to number (time.Duration nanoseconds)

UPDATE workflows 
SET config = json_replace(
    config,
    '$.TestnetDuration',
    CASE 
        WHEN json_type(config, '$.TestnetDuration') = 'text' THEN
            CASE
                -- Empty string becomes 0
                WHEN json_extract(config, '$.TestnetDuration') = '' THEN 0
                -- Parse Go duration strings back to nanoseconds
                WHEN json_extract(config, '$.TestnetDuration') LIKE '%h' THEN
                    CAST(REPLACE(json_extract(config, '$.TestnetDuration'), 'h', '') AS INTEGER) * 3600000000000
                WHEN json_extract(config, '$.TestnetDuration') LIKE '%m' THEN
                    CAST(REPLACE(json_extract(config, '$.TestnetDuration'), 'm', '') AS INTEGER) * 60000000000
                WHEN json_extract(config, '$.TestnetDuration') LIKE '%s' THEN
                    CAST(REPLACE(json_extract(config, '$.TestnetDuration'), 's', '') AS INTEGER) * 1000000000
                ELSE 0
            END
        ELSE json_extract(config, '$.TestnetDuration')
    END
)
WHERE json_type(config, '$.TestnetDuration') IS NOT NULL; 