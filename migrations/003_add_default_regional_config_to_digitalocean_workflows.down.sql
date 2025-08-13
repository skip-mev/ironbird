-- Revert the addition of default regional config to DigitalOcean workflows
-- This removes the nyc1 regional config that was added by the up migration

UPDATE workflows 
SET config = json_replace(
    config,
    '$.ChainConfig.RegionConfigs',
    json_array()
)
WHERE 
    json_extract(config, '$.RunnerType') = 'DigitalOcean'
    AND 
    json_array_length(json_extract(config, '$.ChainConfig.RegionConfigs')) = 1
    AND
    json_extract(config, '$.ChainConfig.RegionConfigs[0].name') = 'nyc1'
    AND
    json_extract(config, '$.ChainConfig.NumOfValidators') > 0;
