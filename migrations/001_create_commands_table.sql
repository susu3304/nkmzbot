-- Create commands table
CREATE TABLE IF NOT EXISTS commands (
    guild_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    response TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (guild_id, name)
);

-- Create index for better performance
CREATE INDEX IF NOT EXISTS idx_commands_guild_id ON commands(guild_id);
