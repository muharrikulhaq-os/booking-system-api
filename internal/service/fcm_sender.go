package service

import (
	"context"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FCMSender mengirim push notification via Firebase Cloud Messaging sehingga
// notifikasi tetap sampai walau aplikasi ditutup (seperti WhatsApp).
//
// Bila file kredensial service-account tidak ada, konstruktor mengembalikan
// nil dan push FCM dinonaktifkan — WebSocket realtime tetap berjalan.
type FCMSender struct {
	client *messaging.Client
}

func NewFCMSender(credFile string) *FCMSender {
	if _, err := os.Stat(credFile); err != nil {
		log.Printf("FCM nonaktif: file kredensial %q tidak ditemukan", credFile)
		return nil
	}
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credFile))
	if err != nil {
		log.Printf("FCM nonaktif: gagal init Firebase app: %v", err)
		return nil
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		log.Printf("FCM nonaktif: gagal init Messaging client: %v", err)
		return nil
	}
	log.Printf("FCM aktif (kredensial: %s)", credFile)
	return &FCMSender{client: client}
}

// Send mengirim satu pesan ke satu device token. `unregistered` true bila FCM
// menyatakan token sudah mati (app di-uninstall / token rotasi) — pemanggil
// sebaiknya menghapus token tsb dari DB.
func (f *FCMSender) Send(
	ctx context.Context,
	token, title, body string,
	data map[string]string,
) (unregistered bool, err error) {
	msg := &messaging.Message{
		Token:        token,
		Notification: &messaging.Notification{Title: title, Body: body},
		Data:         data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ChannelID: "reservation_notif", // channel yang dibuat app Flutter
			},
		},
	}
	_, err = f.client.Send(ctx, msg)
	if err != nil && messaging.IsUnregistered(err) {
		return true, err
	}
	return false, err
}
