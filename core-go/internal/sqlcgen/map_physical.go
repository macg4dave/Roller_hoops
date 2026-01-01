package sqlcgen

import (
	"context"
	"time"
)

type MapDeviceLinkPeer struct {
	LinkID          string
	LinkKey         string
	PeerDeviceID    string
	PeerDisplayName *string
	LinkType        *string
	Source          string
	LastSeenAt      time.Time
}

const listDeviceLinkPeers = `-- name: ListDeviceLinkPeers :many
WITH links_with_peers AS (
  SELECT l.id::text AS link_id,
         l.link_key,
         CASE
           WHEN l.a_device_id = $1::uuid THEN l.b_device_id
           ELSE l.a_device_id
         END AS peer_device_uuid,
         CASE
           WHEN l.a_device_id = $1::uuid THEN l.b_device_id::text
           ELSE l.a_device_id::text
         END AS peer_device_id,
         l.link_type,
         l.source,
         COALESCE(l.observed_at, l.updated_at) AS last_seen_at
  FROM links l
  WHERE l.a_device_id = $1::uuid OR l.b_device_id = $1::uuid
)
SELECT DISTINCT ON (l.peer_device_id)
       l.link_id,
       l.link_key,
       l.peer_device_id,
       d.display_name AS peer_display_name,
       l.link_type,
       l.source,
       l.last_seen_at
FROM links_with_peers l
JOIN devices d ON d.id = l.peer_device_uuid
ORDER BY l.peer_device_id ASC, l.link_id ASC
LIMIT $2;
`

func (q *Queries) ListDeviceLinkPeers(ctx context.Context, deviceID string, limit int32) ([]MapDeviceLinkPeer, error) {
	rows, err := q.db.Query(ctx, listDeviceLinkPeers, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MapDeviceLinkPeer
	for rows.Next() {
		var i MapDeviceLinkPeer
		if err := rows.Scan(
			&i.LinkID,
			&i.LinkKey,
			&i.PeerDeviceID,
			&i.PeerDisplayName,
			&i.LinkType,
			&i.Source,
			&i.LastSeenAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

