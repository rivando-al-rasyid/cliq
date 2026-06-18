CREATE TABLE profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    full_name varchar(100) NULL,
    phone varchar(30) NULL,
    photo text NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NULL
);
