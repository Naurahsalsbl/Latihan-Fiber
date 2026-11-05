package service 

import ( 
	"context" 
	"encoding/json" 
	"errors" 
	"fmt" 
	"log/slog" 
	"github.com/hibiken/asynq" 
	"latihan-fiber/modul4/app/model" 
	"latihan-fiber/modul4/app/repository" 
	"latihan-fiber/modul4/helper" 
) 

// NotificationWorker mengerjakan task "notification:send". 
// 
// Bagi worker, Asynq adalah framework-nya — sama seperti Fiber bagi API. 
// Karena itu method di sini menerima *asynq.Task, seperti method service 
// lain menerima *fiber.Ctx. 
type NotificationWorker struct { 
	users repository.UserRepository 
	notifications repository.NotificationRepository 
	tx *repository.TxManager 
	mailer helper.Mailer 
	logger *slog.Logger 
} 

func NewNotificationWorker( 
	users repository.UserRepository, 
	notifications repository.NotificationRepository, 
	tx *repository.TxManager, 
	mailer helper.Mailer, 
	logger *slog.Logger, 
) *NotificationWorker { 
	return &NotificationWorker{users: users, notifications: notifications, 
		tx: tx, mailer: mailer, logger: logger} 
}

// HandleSend mengembalikan error untuk meminta Asynq melakukan retry nanti, 
// atau error yang membungkus asynq.SkipRetry untuk menyatakan "jangan di-retry, 
// task ini tidak mungkin berhasil". 
// 
// Asynq menjamin setiap task dijalankan PALING SEDIKIT satu kali, bukan 
// TEPAT satu kali. Task yang sama bisa datang dua kali, misalnya bila 
// worker mati setelah mengirim email tetapi sebelum sempat melapor ke Redis. 
// Karena itu handler ini harus idempoten: menjalankannya dua kali untuk 
// task yang sama harus berakibat sama dengan menjalankannya sekali. 
func (w *NotificationWorker) HandleSend(ctx context.Context, t *asynq.Task) error { 
	var job model.NotificationJob 
	if err := json.Unmarshal(t.Payload(), &job); err != nil { 
		return fmt.Errorf("%w: payload task rusak: %v", asynq.SkipRetry, err) 
	} 
	var event model.EnrollmentEvent 
	if err := json.Unmarshal(job.Payload, &event); err != nil { 
		return fmt.Errorf("%w: payload event %d rusak: %v", asynq.SkipRetry, job.OutboxID, err) 
	} 

	if job.EventType == model.EventCourseQuotaChanged {
		return w.handleQuotaChanged(ctx, job)
	}
	
	user, err := w.users.FindByID(ctx, event.UserID) 
	if errors.Is(err, repository.ErrNotFound) { 
		// Akunnya sudah dihapus. Melakukan retry tidak akan memunculkannya. 
		return fmt.Errorf("%w: user %d tidak ditemukan", asynq.SkipRetry, event.UserID)
		} 
	if err != nil { 
		return err // database sedang bermasalah: layak di-retry 
	} 
		
	msg, err := BuildMessage(job.EventType, event, user) 
	if err != nil { 
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err) 
	} 
		
	alreadySent := false 
	err = w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error { 
		// Claim dan pengiriman berada di dalam SATU transaction: 
		// - pengiriman gagal -> claim ikut dibatalkan -> attempt 
		// berikutnya masih dapat mengirim; 
		// - pengiriman sukses -> claim di-commit -> attempt berikutnya 
		// melihat claim ini dan tidak mengirim lagi. 
		claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{ 
			OutboxID: job.OutboxID, UserID: user.ID, Channel: "email", 
			Subject: msg.Subject, Body: msg.Body, 
		}) 
		if err != nil { 
			return err 
		} 
		if !claimed { 
			alreadySent = true 
			return nil 
		} 
		return w.mailer.Send(ctx, msg) 
	}) 
		
	if errors.Is(err, helper.ErrPermanentDelivery) { 
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err) 
	} 
	if err != nil { 
		return err 
	} 
		
	retry, _ := asynq.GetRetryCount(ctx) 
	w.logger.Info("notification_processed", 
		slog.Int64("outbox_id", job.OutboxID), 
		slog.String("event", job.EventType), 
		slog.Int("retry", retry), slog.Bool("already_sent", alreadySent)) 
	return nil 
}

// handleQuotaChanged mengirim satu email per peserta. Setiap penerima punya
// claim sendiri (outbox_id, user_id) dalam transaction-nya sendiri, jadi
// pengiriman yang sudah commit TIDAK diulang saat task di-retry.
func (w *NotificationWorker) handleQuotaChanged(ctx context.Context, job model.NotificationJob) error {
	var event model.QuotaChangedEvent
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return fmt.Errorf("%w: payload event %d rusak: %v", asynq.SkipRetry, job.OutboxID, err)
	}

	var firstTemp error
	sent, skipped, dropped, failed := 0, 0, 0, 0

	for _, userID := range event.UserIDs {
		if err := ctx.Err(); err != nil {
			return err
		}
		alreadySent, err := w.sendQuotaMail(ctx, job.OutboxID, userID, event)
		switch {
		case err == nil && alreadySent:
			skipped++
		case err == nil:
			sent++
		case errors.Is(err, repository.ErrNotFound), errors.Is(err, helper.ErrPermanentDelivery):
			// Penerima ini tidak mungkin berhasil; jangan menahan penerima lain.
			w.logger.Warn("quota_recipient_dropped",
				slog.Int64("outbox_id", job.OutboxID), slog.Int("user_id", userID),
				slog.String("error", err.Error()))
			dropped++
		default:
			failed++
			if firstTemp == nil {
				firstTemp = err
			}
		}
	}

	// Satu kegagalan sementara tidak menghentikan penerima lain, tetapi task
	// tetap dikembalikan ke Asynq agar yang gagal dicoba lagi.
	if firstTemp != nil {
		return fmt.Errorf("%d dari %d penerima gagal dikirimi: %w",
			failed, len(event.UserIDs), firstTemp)
	}

	retry, _ := asynq.GetRetryCount(ctx)
	w.logger.Info("notification_processed",
		slog.Int64("outbox_id", job.OutboxID),
		slog.String("event", job.EventType),
		slog.Int("retry", retry),
		slog.Bool("already_sent", sent == 0),
		slog.Int("sent", sent), slog.Int("skipped", skipped), slog.Int("dropped", dropped))
	return nil
}

func (w *NotificationWorker) sendQuotaMail(ctx context.Context, outboxID int64, userID int, event model.QuotaChangedEvent) (bool, error) {
	user, err := w.users.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}
	msg := BuildQuotaMessage(event, user)

	alreadySent := false
	err = w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
		claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
			OutboxID: outboxID, UserID: user.ID, Channel: "email",
			Subject: msg.Subject, Body: msg.Body,
		})
		if err != nil {
			return err
		}
		if !claimed {
			alreadySent = true
			return nil
		}
		return w.mailer.Send(ctx, msg)
	})
	return alreadySent, err
}