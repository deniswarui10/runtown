# Authboss Migration Guide

This guide covers the migration of existing users to the new Authboss authentication system.

## Overview

The Authboss migration process ensures that existing users can seamlessly use the new authentication system without losing their accounts or having to re-register.

## Migration Scripts

### 1. Verification Script
**Command:** `go run ./cmd/verify-authboss`

Checks the current migration status and verifies:
- ✅ All required Authboss columns exist
- 📊 User migration completion percentage
- 📊 Number of confirmed vs unconfirmed users
- 📊 Currently locked accounts
- 📊 Users with failed login attempts
- ✅ Remember tokens table status

### 2. Migration Script
**Command:** `go run ./cmd/migrate-authboss`

Performs the actual migration:
- ✅ Verifies prerequisites before migration
- 🔄 Migrates users in batches with transaction safety
- ✅ Sets appropriate default values for Authboss fields
- ✅ Preserves existing user data and relationships
- ✅ Verifies migration success

### 3. Rollback Script
**Command:** `go run ./cmd/rollback-authboss`

Rolls back the migration if needed:
- ⚠️ Removes Authboss-specific data
- ✅ Preserves core user data (email, password, names, etc.)
- 🧹 Clears remember tokens
- ✅ Safe transaction-based rollback

## Migration Process

### Step 1: Pre-Migration Verification
```bash
# Check current status
go run ./cmd/verify-authboss
```

### Step 2: Backup Database
```bash
# Create a backup before migration
pg_dump your_database > backup_before_authboss_migration.sql
```

### Step 3: Run Database Migrations
```bash
# Ensure all Authboss columns exist
go run ./cmd/migrate
```

### Step 4: Run User Data Migration
```bash
# Migrate existing users to Authboss format
go run ./cmd/migrate-authboss
```

### Step 5: Post-Migration Verification
```bash
# Verify migration was successful
go run ./cmd/verify-authboss
```

### Step 6: Test Authentication
```bash
# Start the new Authboss-enabled server
go run ./cmd/server2
```

## Migration Details

### What Gets Migrated

**Preserved Data:**
- ✅ User ID, email, password hash
- ✅ First name, last name, role
- ✅ Creation and update timestamps
- ✅ Email verification status

**New Authboss Fields:**
- `confirmed_at` - Set based on existing email verification
- `attempt_count` - Reset to 0 for all users
- `locked_until` - NULL (no users locked initially)
- `last_attempt` - NULL (no previous failed attempts)
- `password_changed_at` - Set to user creation date
- Token fields - NULL (will be generated as needed)

### Migration Safety Features

**Transaction Safety:**
- All migrations run in database transactions
- Automatic rollback on any error
- Batch processing for large user bases

**Verification:**
- Pre-migration prerequisite checks
- Post-migration success verification
- Detailed logging and progress reporting

**Rollback Capability:**
- Complete rollback script available
- Preserves core user data
- Safe to run multiple times

## Troubleshooting

### Common Issues

**1. Missing Authboss Columns**
```
Error: required Authboss column 'confirmed_at' does not exist
```
**Solution:** Run database migrations first: `go run ./cmd/migrate`

**2. Migration Already Complete**
```
Warning: X users already have Authboss data
```
**Solution:** This is normal. The migration will update existing data safely.

**3. Database Connection Issues**
```
Error: Failed to connect to database
```
**Solution:** Check database configuration in your environment variables.

### Verification Commands

```bash
# Check if migrations are needed
go run ./cmd/verify-authboss

# Check database schema
psql -d your_database -c "\d users"

# Count migrated users
psql -d your_database -c "SELECT COUNT(*) FROM users WHERE attempt_count IS NOT NULL;"
```

## Post-Migration Steps

### 1. Update Server Configuration
Switch from the old server to the new Authboss-enabled server:
```bash
# Old server
go run ./cmd/server

# New Authboss server
go run ./cmd/server2
```

### 2. Test Authentication Flows
- ✅ Login with existing users
- ✅ Registration of new users
- ✅ Password reset functionality
- ✅ Account locking after failed attempts
- ✅ Remember me functionality

### 3. Monitor Security Events
The new system logs all authentication events:
```
[AUTHBOSS INFO] Security Event: login_success | Email: user@example.com | IP: 127.0.0.1 | ...
```

### 4. Update Documentation
Update any authentication-related documentation to reflect the new Authboss system.

## Rollback Procedure

If you need to rollback the migration:

### Step 1: Stop the Authboss Server
```bash
# Stop the new server
# Start the old server: go run ./cmd/server
```

### Step 2: Run Rollback Script
```bash
go run ./cmd/rollback-authboss
```

### Step 3: Verify Rollback
```bash
go run ./cmd/verify-authboss
```

### Step 4: Restore from Backup (if needed)
```bash
# If complete restoration is needed
psql your_database < backup_before_authboss_migration.sql
```

## Security Considerations

### Enhanced Security Features
The new Authboss system provides:
- 🔒 Account locking after failed attempts
- 📊 Comprehensive security event logging
- 🔐 Strong password validation
- 🔄 Secure session management
- 🎫 Remember me tokens

### Migration Security
- All password hashes are preserved unchanged
- No plaintext passwords are ever exposed
- Transaction-based migration ensures data integrity
- Rollback capability provides safety net

## Support

If you encounter issues during migration:
1. Check the troubleshooting section above
2. Review the migration logs for specific error messages
3. Verify database connectivity and permissions
4. Ensure all prerequisites are met

The migration scripts are designed to be safe and can be run multiple times without causing issues.