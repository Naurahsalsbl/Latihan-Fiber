// Alat uji untuk C.5 (idempotency).
//
// Membaca satu event dari tabel outbox, lalu memasukkan N salinan task
// notifikasi untuk event itu langsung ke Redis, melewati relay.
//
// Salinan sengaja dibuat TANPA TaskID. Bila diberi TaskID "outbox-<id>"
// seperti milik relay, Redis akan menolak salinan kedua dan seterusnya
// (ErrTaskIDConflict), sehingga idempotency worker tidak pernah teruji.
//
// Pemakaian (dari folder proyek, tempat .env berada):
//
//	go run ./tools/redeliver -outbox 42 -copies 5
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"

	"latihan-fiber/modul4/app/model"
	"latihan-fiber/modul4/config"
	"latihan-fiber/modul4/database"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "redeliver gagal:", err)
		os.Exit(1)
	}
}

func run() error {
	outboxID := flag.Int64("outbox", 0, "id baris outbox yang akan dikirim ulang")
	copies := flag.Int("copies", 5, "jumlah salinan task yang dimasukkan ke Redis")
	flag.Parse()

	if *outboxID <= 0 {
		return errors.New("flag -outbox wajib diisi dengan id outbox yang valid")
	}
	if *copies <= 0 {
		return errors.New("flag -copies harus lebih dari 0")
	}

	config.LoadEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx)
	if err != nil {
		return fmt.Errorf("terhubung ke database: %w", err)
	}
	defer pool.Close()

	var (
		eventType   string
		payload     []byte
		publishedAt *time.Time
	)
	err = pool.QueryRow(ctx,
		`SELECT event_type, payload, published_at FROM outbox WHERE id = $1`,
		*outboxID).Scan(&eventType, &payload, &publishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("outbox dengan id %d tidak ditemukan", *outboxID)
	}
	if err != nil {
		return fmt.Errorf("membaca outbox: %w", err)
	}

	if publishedAt != nil {
		fmt.Printf("PERINGATAN: outbox %d sudah di-publish (%s). "+
			"Untuk C.5, event seharusnya masih tertahan (published_at NULL).\n",
			*outboxID, publishedAt.Format(time.RFC3339))
	}

	body, err := json.Marshal(model.NotificationJob{
		OutboxID:  *outboxID,
		EventType: eventType,
		Payload:   json.RawMessage(payload),
	})
	if err != nil {
		return fmt.Errorf("menyusun payload task: %w", err)
	}

	client := asynq.NewClient(config.RedisOpt())
	defer client.Close()

	for i := 1; i <= *copies; i++ {
		task := asynq.NewTask(model.TaskSendNotification, body,
			asynq.Queue(model.QueueNotifications),
			asynq.MaxRetry(5),
			asynq.Timeout(30*time.Second),
			asynq.Retention(24*time.Hour),
		)
		info, err := client.EnqueueContext(ctx, task)
		if err != nil {
			return fmt.Errorf("enqueue salinan ke-%d: %w", i, err)
		}
		fmt.Printf("salinan %d/%d dimasukkan: task id=%s queue=%s\n",
			i, *copies, info.ID, info.Queue)
	}

	fmt.Printf("selesai: %d salinan untuk outbox %d (%s). "+
		"Relay akan menambahkan satu lagi begitu worker menyala.\n",
		*copies, *outboxID, eventType)
	return nil
}