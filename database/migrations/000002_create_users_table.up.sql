CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email varchar(255) NOT NULL UNIQUE,
    password text NOT NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NULL
);
