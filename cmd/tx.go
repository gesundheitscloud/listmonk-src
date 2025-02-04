package main

import (
	"errors"
	"net/http"
	"net/textproto"

	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// handleSendTxMessage handles the sending of a transactional message.
func handleSendTxMessage(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		m   models.TxMessage
	)

	if err := c.Bind(&m); err != nil {
		return err
	}

	// Validate input.
	if r, err := validateTxMessage(m, app); err != nil {
		return err
	} else {
		m = r
	}

	// Extract latest tx template from DB.
	dbTpl, err := app.core.GetTemplate(m.TemplateID, true)
	if err != nil {
		return err
	}

	// Compare with cached tx template and update cache if outdated.
	tpl, err := app.manager.GetTpl(m.TemplateID)
	if err != nil || tpl.UpdatedAt != dbTpl.UpdatedAt {
		// Extract full tx template from DB.
		dbTpl, err := app.core.GetTemplate(m.TemplateID, false)
		if err != nil {
			return err
		}

		// Compile the template and validate.
		tpl = &dbTpl
		if err := tpl.Compile(app.manager.GenericTemplateFuncs()); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		app.manager.CacheTpl(m.TemplateID, tpl)
	}

	// Get the subscriber.
	sub, err := app.core.GetSubscriber(m.SubscriberID, "", m.SubscriberEmail)
	if err != nil {
		var httpErr *echo.HTTPError
		// If subscriber email is defined but unknown (StatusBadRequest), we can continue
		if (errors.As(err, &httpErr) && httpErr.Code != http.StatusBadRequest) || m.SubscriberEmail == "" {
			return err
		}
		sub.Email = m.SubscriberEmail
	}

	// Render the message.
	if err := m.Render(sub, tpl); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.errorFetching", "name"))
	}

	// Prepare the final message.
	msg := manager.Message{}
	msg.Subscriber = sub
	msg.To = []string{sub.Email}
	msg.From = m.FromEmail
	msg.Subject = m.Subject
	msg.ContentType = m.ContentType
	msg.Messenger = m.Messenger
	msg.Body = m.Body

	// Optional headers.
	if len(m.Headers) != 0 {
		msg.Headers = make(textproto.MIMEHeader, len(m.Headers))
		for _, set := range m.Headers {
			for hdr, val := range set {
				msg.Headers.Add(hdr, val)
			}
		}
	}

	if err := app.manager.PushMessage(msg); err != nil {
		app.log.Printf("error sending message (%s): %v", msg.Subject, err)
		if errors.Is(err, manager.ErrTimeout) {
			return echo.NewHTTPError(http.StatusTooManyRequests, err.Error())
		}
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func validateTxMessage(m models.TxMessage, app *App) (models.TxMessage, error) {
	if m.SubscriberEmail == "" && m.SubscriberID == 0 {
		return m, echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.missingFields", "name", "subscriber_email or subscriber_id"))
	}

	if m.SubscriberEmail != "" {
		em, err := app.importer.SanitizeEmail(m.SubscriberEmail)
		if err != nil {
			return m, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		m.SubscriberEmail = em
	}

	if m.FromEmail == "" {
		m.FromEmail = app.constants.FromEmail
	}

	if m.Messenger == "" {
		m.Messenger = emailMsgr
	} else if !app.manager.HasMessenger(m.Messenger) {
		return m, echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("campaigns.fieldInvalidMessenger", "name", m.Messenger))
	}

	return m, nil
}
