-- Permissions table
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(resource, action)
);

CREATE INDEX idx_permissions_name ON permissions(name);
CREATE INDEX idx_permissions_resource ON permissions(resource);

-- Role-Permissions junction table
CREATE TABLE role_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, permission_id)
);

CREATE INDEX idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions(permission_id);

-- Activity logs table
CREATE TABLE activity_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    username VARCHAR(50),
    action VARCHAR(50) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    resource_id VARCHAR(100),
    description TEXT NOT NULL,
    ip_address VARCHAR(50),
    user_agent TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_activity_logs_user_id ON activity_logs(user_id);
CREATE INDEX idx_activity_logs_action ON activity_logs(action);
CREATE INDEX idx_activity_logs_resource ON activity_logs(resource);
CREATE INDEX idx_activity_logs_created_at ON activity_logs(created_at);

-- Insert default permissions
INSERT INTO permissions (name, resource, action, description) VALUES
    -- User permissions
    ('users.create', 'users', 'create', 'Create new users'),
    ('users.read', 'users', 'read', 'View users'),
    ('users.update', 'users', 'update', 'Update users'),
    ('users.delete', 'users', 'delete', 'Delete/deactivate users'),
    -- Role permissions
    ('roles.create', 'roles', 'create', 'Create new roles'),
    ('roles.read', 'roles', 'read', 'View roles'),
    ('roles.update', 'roles', 'update', 'Update roles'),
    ('roles.delete', 'roles', 'delete', 'Delete roles'),
    -- Permission permissions
    ('permissions.read', 'permissions', 'read', 'View permissions'),
    ('permissions.assign', 'permissions', 'assign', 'Assign permissions to roles'),
    -- Category permissions
    ('categories.create', 'categories', 'create', 'Create categories'),
    ('categories.read', 'categories', 'read', 'View categories'),
    ('categories.update', 'categories', 'update', 'Update categories'),
    ('categories.delete', 'categories', 'delete', 'Delete categories'),
    -- Product permissions
    ('products.create', 'products', 'create', 'Create products'),
    ('products.read', 'products', 'read', 'View products'),
    ('products.update', 'products', 'update', 'Update products'),
    ('products.delete', 'products', 'delete', 'Delete products'),
    -- Inventory permissions
    ('inventory.read', 'inventory', 'read', 'View inventory'),
    ('inventory.adjust', 'inventory', 'adjust', 'Adjust inventory'),
    -- Customer permissions
    ('customers.create', 'customers', 'create', 'Create customers'),
    ('customers.read', 'customers', 'read', 'View customers'),
    ('customers.update', 'customers', 'update', 'Update customers'),
    ('customers.delete', 'customers', 'delete', 'Delete customers'),
    -- Sale permissions
    ('sales.create', 'sales', 'create', 'Create sales'),
    ('sales.read', 'sales', 'read', 'View sales'),
    ('sales.cancel', 'sales', 'cancel', 'Cancel sales'),
    ('sales.refund', 'sales', 'refund', 'Refund sales'),
    -- Payment permissions
    ('payments.create', 'payments', 'create', 'Process payments'),
    ('payments.read', 'payments', 'read', 'View payments'),
    -- Cashier permissions
    ('cashier.open_shift', 'cashier', 'open_shift', 'Open cashier shift'),
    ('cashier.close_shift', 'cashier', 'close_shift', 'Close cashier shift'),
    ('cashier.view_shift', 'cashier', 'view_shift', 'View shift info'),
    -- Report permissions
    ('reports.read', 'reports', 'read', 'View reports'),
    -- Activity log permissions
    ('activity_logs.read', 'activity_logs', 'read', 'View activity logs');

-- Assign all permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin';

-- Assign manager permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'manager'
AND p.name IN (
    'categories.create', 'categories.read', 'categories.update', 'categories.delete',
    'products.create', 'products.read', 'products.update', 'products.delete',
    'inventory.read', 'inventory.adjust',
    'customers.create', 'customers.read', 'customers.update', 'customers.delete',
    'sales.create', 'sales.read', 'sales.cancel', 'sales.refund',
    'payments.create', 'payments.read',
    'reports.read',
    'activity_logs.read'
);

-- Assign cashier permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'cashier'
AND p.name IN (
    'products.read',
    'customers.create', 'customers.read', 'customers.update',
    'sales.create', 'sales.read',
    'payments.create',
    'cashier.open_shift', 'cashier.close_shift', 'cashier.view_shift'
);
