-- Add default regional config (nyc1) to DigitalOcean workflows that don't have one
-- This migration addresses existing DigitalOcean workflows that were created before 
-- regional configuration was implemented

UPDATE workflows 
SET config = json_replace(
    config,
    '$.ChainConfig.RegionConfigs',
    json_array(
        json_object(
            'name', 'nyc1',
            'num_validators', json_array_length(validators),
            'num_nodes', json_array_length(nodes)
        )
    )
)
WHERE 
    json_extract(config, '$.RunnerType') = 'DigitalOcean'
    AND (
        json_extract(config, '$.ChainConfig.RegionConfigs') IS NULL
        OR
        json_array_length(json_extract(config, '$.ChainConfig.RegionConfigs')) = 0
    )
