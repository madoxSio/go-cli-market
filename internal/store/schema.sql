-- Comptes et sessions. Le rôle admin est prévu pour la gestion des utilisateurs,
-- l'inscription publique crée toujours un client.

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'client' CHECK (role IN ('client', 'admin')),
    confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    confirmation_code TEXT,
    confirmation_expires_at TIMESTAMPTZ,
    reset_code TEXT,
    reset_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions (user_id);

-- Catalogue. Le prix TTC n'est pas stocké : il se calcule avec market.PriceWithVAT.

CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    ref TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    description TEXT NOT NULL DEFAULT '' CHECK (char_length(description) <= 2000),
    category TEXT NOT NULL CHECK (char_length(category) BETWEEN 1 AND 100),
    price_ht BIGINT NOT NULL CHECK (price_ht > 0),
    vat_rate INTEGER NOT NULL CHECK (vat_rate BETWEEN 0 AND 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS products_category_idx ON products (category);

-- Paniers. Un client a au plus un panier « open » ; payé, il passe « checked_out ».

CREATE TABLE IF NOT EXISTS carts (
    id BIGSERIAL PRIMARY KEY,
    ref TEXT NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'checked_out')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS carts_one_open_per_user ON carts (user_id) WHERE status = 'open';

CREATE TABLE IF NOT EXISTS cart_items (
    cart_id BIGINT NOT NULL REFERENCES carts (id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products (id),
    quantity INTEGER NOT NULL CHECK (quantity BETWEEN 1 AND 99),
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (cart_id, product_id)
);

-- Commandes. Les lignes copient le nom et les prix du produit au moment de la commande.

CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    ref TEXT NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    cart_id BIGINT REFERENCES carts (id) ON DELETE SET NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'paid', 'shipping', 'delivered', 'cancelled')),
    cancel_reason TEXT NOT NULL DEFAULT '',
    total_ttc BIGINT NOT NULL CHECK (total_ttc >= 0),
    card_last4 TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (status <> 'cancelled' OR cancel_reason <> '')
);

CREATE INDEX IF NOT EXISTS orders_user_id_idx ON orders (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS order_items (
    order_id BIGINT NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    product_id BIGINT REFERENCES products (id) ON DELETE SET NULL,
    product_ref TEXT NOT NULL,
    name TEXT NOT NULL,
    unit_price_ht BIGINT NOT NULL,
    vat_rate INTEGER NOT NULL,
    unit_price_ttc BIGINT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (order_id, position)
);
