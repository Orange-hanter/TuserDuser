-- Создание таблицы events
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(100) NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    duration INTEGER NOT NULL, -- в минутах
    place VARCHAR(255),
    price_type VARCHAR(50) NOT NULL DEFAULT 'free',
    need_registration BOOLEAN NOT NULL DEFAULT false,
    details JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_start_time ON events(start_time);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);

-- Вставка тестовых данных
INSERT INTO events (type, start_time, end_time, duration, place, price_type, need_registration, details)
VALUES 
    ('Встреча', '2025-10-10T18:05:00Z', '2025-10-10T18:35:00Z', 30, 'Офис', 'free', false, '{"description": "Встреча команды"}'),
    ('Конференция', '2025-11-15T09:00:00Z', '2025-11-15T18:00:00Z', 540, 'Конференц-зал', 'paid', true, '{"price": 1500, "capacity": 100}'),
    ('Вебинар', '2025-10-20T14:00:00Z', '2025-10-20T15:30:00Z', 90, 'Online', 'free', true, '{"link": "https://zoom.us/meeting"}');
