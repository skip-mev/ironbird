-- Convert TestnetDuration from number to string in workflow configs
-- This migration handles the type change from time.Duration to string in
--- https://github.com/skip-mev/ironbird/commit/806ddeb9c37b134dbdba29fa983e3244f57c9205#diff-0da9aa2f11acde9fd3c852522abe02434e196a252b4b1d051981556f3c276c40R79

UPDATE workflows 
SET config = json_replace(
    config,
    '$.TestnetDuration',
    CASE 
        -- If TestnetDuration is a number (duration in nanoseconds), convert to Go duration string
        WHEN json_type(config, '$.TestnetDuration') = 'integer' THEN
            CASE
                WHEN json_extract(config, '$.TestnetDuration') = 0 THEN ''
                WHEN json_extract(config, '$.TestnetDuration') >= 3600000000000 THEN 
                    -- 1 hour or more: convert to hours (e.g., "2h")
                    (json_extract(config, '$.TestnetDuration') / 3600000000000) || 'h'
                WHEN json_extract(config, '$.TestnetDuration') >= 60000000000 THEN 
                    -- 1 minute or more: convert to minutes (e.g., "30m")
                    (json_extract(config, '$.TestnetDuration') / 60000000000) || 'm'
                ELSE 
                    -- Less than 1 minute: convert to seconds (e.g., "30s")
                    (json_extract(config, '$.TestnetDuration') / 1000000000) || 's'
            END
        -- If already a string, leave it as is
        ELSE json_extract(config, '$.TestnetDuration')
    END
)
WHERE json_type(config, '$.TestnetDuration') IS NOT NULL; 