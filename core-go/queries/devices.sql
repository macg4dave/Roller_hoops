-- name: ListDevices :many
+SELECT id, display_name
+FROM devices
+ORDER BY created_at DESC;
+
+-- name: GetDevice :one
+SELECT id, display_name
+FROM devices
+WHERE id = $1;
+
+-- name: CreateDevice :one
+INSERT INTO devices (display_name)
+VALUES ($1)
+RETURNING id, display_name;
+
+-- name: UpdateDevice :one
+UPDATE devices
+SET display_name = $2,
+    updated_at = now()
+WHERE id = $1
+RETURNING id, display_name;
