const NOTIFICATION_SERVER_URL =
  process.env.NOTIFICATION_SERVER_URL || "https://aurahub-go-notifier.onrender.com";

/**
 * Send real-time notification to the Aurahub Go Notifier service.
 * @param {string|object} recipientId - The user receiving the notification
 * @param {object} notification - The populated notification document
 */
export async function sendRealtimeNotification(recipientId, notification) {
  if (!recipientId || !notification) return;

  try {
    const url = `${NOTIFICATION_SERVER_URL.replace(/\/$/, "")}/api/notify`;
    await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        recipientId: recipientId.toString(),
        notification,
      }),
      signal: AbortSignal.timeout(4000),
    });
  } catch {
    // Non-blocking: If Go Notifier is temporarily spinning up on Render free tier or offline,
    // the notification is already persisted in MongoDB so the user will still receive it.
  }
}
