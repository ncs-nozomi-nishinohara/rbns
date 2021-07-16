package repository

import "context"

type Repository interface {
	NewConnection() Connection
}

type Connection interface {
	Permission(ctx context.Context) Permission
	Role(ctx context.Context) Role
	Organization(ctx context.Context) Organization
	User(ctx context.Context) User
	Transaction(ctx context.Context) Tx
}

type Tx interface {
	Do(fn func(tx Transaction) error) error
}

type Transaction interface {
	Permission() PermissionCommand
	Role() RoleCommand
	Organization() OrganizationCommand
	User() UserCommand
}
