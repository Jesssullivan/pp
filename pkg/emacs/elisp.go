package emacs

// GenerateElispPackage returns the contents of prompt-pulse.el,
// a complete Emacs package that can be installed via use-package.
// It provides dashboard integration with auto-refresh, waifu image support,
// and hooks for eat-mode, doom-dashboard, and enlight.
func GenerateElispPackage() string {
	return `;;; prompt-pulse.el --- Prompt Pulse dashboard integration -*- lexical-binding: t; -*-

;; Author: Tinyland Infrastructure
;; Version: 1.0.0
;; Package-Requires: ((emacs "28.1"))
;; Keywords: tools, dashboard
;; URL: https://gitlab.com/tinyland/lab/prompt-pulse

;;; Commentary:

;; This package integrates prompt-pulse system monitoring into Emacs.
;; It provides a dashboard buffer with auto-refresh, widget rendering,
;; and optional waifu image display.
;;
;; Quick start:
;;   (require 'prompt-pulse)
;;   (prompt-pulse-mode 1)
;;
;; Or with use-package:
;;   (use-package prompt-pulse
;;     :config
;;     (prompt-pulse-mode 1))

;;; Code:

(require 'json)

;;;; Customization

(defgroup prompt-pulse nil
  "Prompt Pulse dashboard integration."
  :group 'tools
  :prefix "prompt-pulse-")

(defcustom prompt-pulse-binary-path "prompt-pulse"
  "Path to the prompt-pulse binary."
  :type 'string
  :group 'prompt-pulse)

(defcustom prompt-pulse-refresh-interval 30
  "Dashboard refresh interval in seconds."
  :type 'integer
  :group 'prompt-pulse)

(defcustom prompt-pulse-show-waifu t
  "Whether to display the waifu image in GUI Emacs."
  :type 'boolean
  :group 'prompt-pulse)

(defcustom prompt-pulse-cache-dir
  (expand-file-name "prompt-pulse" (or (getenv "XDG_CACHE_HOME")
                                       (expand-file-name ".cache" "~")))
  "Directory where prompt-pulse stores cached data."
  :type 'directory
  :group 'prompt-pulse)

;;;; Internal variables

(defvar prompt-pulse--timer nil
  "Timer for auto-refresh.")

(defvar prompt-pulse--buffer-name "*prompt-pulse*"
  "Name of the dashboard buffer.")

(defvar prompt-pulse--last-data nil
  "Last parsed JSON data from prompt-pulse.")

;;;; Faces

(defface prompt-pulse-title
  '((t :inherit bold :height 1.2))
  "Face for widget titles."
  :group 'prompt-pulse)

(defface prompt-pulse-ok
  '((t :inherit success))
  "Face for OK status indicators."
  :group 'prompt-pulse)

(defface prompt-pulse-warning
  '((t :inherit warning))
  "Face for warning status indicators."
  :group 'prompt-pulse)

(defface prompt-pulse-error
  '((t :inherit error))
  "Face for error status indicators."
  :group 'prompt-pulse)

;;;; Core functions

(defun prompt-pulse--get-json ()
  "Call prompt-pulse binary and return parsed JSON."
  (let ((output (with-output-to-string
                  (with-current-buffer standard-output
                    (call-process prompt-pulse-binary-path nil t nil
                                 "emacs" "--json"
                                 "--cache-dir" prompt-pulse-cache-dir)))))
    (if (string-empty-p output)
        nil
      (condition-case nil
          (json-parse-string output :object-type 'alist)
        (error nil)))))

(defun prompt-pulse--status-face (status)
  "Return the face for STATUS string."
  (pcase status
    ("ok" 'prompt-pulse-ok)
    ("warning" 'prompt-pulse-warning)
    ("error" 'prompt-pulse-error)
    (_ 'default)))

(defun prompt-pulse--status-icon (status)
  "Return a status icon string for STATUS."
  (pcase status
    ("ok" "[OK]")
    ("warning" "[WARN]")
    ("error" "[ERR]")
    (_ "[??]")))

(defun prompt-pulse--render-widget (widget)
  "Render a single WIDGET alist into the current buffer."
  (let* ((title (alist-get 'title widget ""))
         (status (alist-get 'status widget "unknown"))
         (summary (alist-get 'summary widget "")))
    (insert (propertize title 'face 'prompt-pulse-title) " ")
    (insert (propertize (prompt-pulse--status-icon status)
                        'face (prompt-pulse--status-face status)) " ")
    (insert (propertize summary 'face 'font-lock-string-face))
    (insert "\n")))

;;;###autoload
(defun prompt-pulse-dashboard-insert ()
  "Insert the prompt-pulse dashboard into the current buffer.
Fetches fresh data from the prompt-pulse binary via JSON output."
  (interactive)
  (let ((data (prompt-pulse--get-json)))
    (setq prompt-pulse--last-data data)
    (if (null data)
        (insert (propertize "prompt-pulse: no data available\n" 'face 'font-lock-comment-face))
      (let ((widgets (alist-get 'widgets data))
            (version (alist-get 'version data ""))
            (timestamp (alist-get 'timestamp data ""))
            (waifu-path (alist-get 'waifu_path data)))
        ;; Header
        (insert (propertize (format "Prompt Pulse v%s" version) 'face 'bold))
        (insert "  ")
        (insert (propertize timestamp 'face 'font-lock-comment-face))
        (insert "\n")
        (insert (make-string 40 ?-) "\n")
        ;; Widgets
        (when (vectorp widgets)
          (mapc #'prompt-pulse--render-widget (append widgets nil)))
        ;; Waifu
        (when (and prompt-pulse-show-waifu
                   (display-graphic-p)
                   waifu-path
                   (file-exists-p waifu-path))
          (insert "\n")
          (prompt-pulse-waifu-insert waifu-path))))))

;;;###autoload
(defun prompt-pulse-refresh ()
  "Refresh the prompt-pulse dashboard buffer."
  (interactive)
  (let ((buf (get-buffer-create prompt-pulse--buffer-name)))
    (with-current-buffer buf
      (let ((inhibit-read-only t))
        (erase-buffer)
        (prompt-pulse-dashboard-insert)
        (goto-char (point-min))))
    (display-buffer buf)))

;;;###autoload
(defun prompt-pulse-waifu-insert (path)
  "Insert the waifu image at PATH into the current buffer.
Only works in GUI Emacs with image support."
  (interactive "fWaifu image path: ")
  (if (and (display-graphic-p)
           (image-type-available-p 'png))
      (let ((img (create-image path 'png nil :max-width 200 :max-height 200)))
        (insert-image img "[waifu]")
        (insert "\n"))
    (insert (propertize "[waifu: GUI required]\n" 'face 'font-lock-comment-face))))

;;;; Minor mode

(defun prompt-pulse--start-timer ()
  "Start the auto-refresh timer."
  (when prompt-pulse--timer
    (cancel-timer prompt-pulse--timer))
  (setq prompt-pulse--timer
        (run-with-timer prompt-pulse-refresh-interval
                        prompt-pulse-refresh-interval
                        #'prompt-pulse--auto-refresh)))

(defun prompt-pulse--stop-timer ()
  "Stop the auto-refresh timer."
  (when prompt-pulse--timer
    (cancel-timer prompt-pulse--timer)
    (setq prompt-pulse--timer nil)))

(defun prompt-pulse--auto-refresh ()
  "Auto-refresh the dashboard if the buffer exists."
  (when (get-buffer prompt-pulse--buffer-name)
    (prompt-pulse-refresh)))

;;;###autoload
(define-minor-mode prompt-pulse-mode
  "Toggle Prompt Pulse dashboard auto-refresh.
When enabled, the dashboard buffer is refreshed at
` + "`" + `prompt-pulse-refresh-interval` + "`" + ` seconds."
  :global t
  :lighter " PP"
  :group 'prompt-pulse
  (if prompt-pulse-mode
      (progn
        (prompt-pulse--start-timer)
        (prompt-pulse-refresh))
    (prompt-pulse--stop-timer)))

;;;; Integration hooks

(defun prompt-pulse-eat-mode-hook ()
  "Hook for eat-mode terminal integration.
Adds prompt-pulse info to the terminal mode line."
  (when prompt-pulse--last-data
    (let* ((widgets (alist-get 'widgets prompt-pulse--last-data))
           (summaries (when (vectorp widgets)
                        (mapcar (lambda (w) (alist-get 'summary w ""))
                                (append widgets nil)))))
      (setq-local mode-line-misc-info
                  (list (string-join (or summaries '("--")) " | "))))))

(defun prompt-pulse-doom-dashboard-section ()
  "Section function for doom-dashboard integration.
Add to ` + "`" + `+doom-dashboard-functions` + "`" + ` to display prompt-pulse."
  (insert "\n")
  (prompt-pulse-dashboard-insert)
  (insert "\n"))

(defun prompt-pulse-enlight-widget ()
  "Widget function for enlight dashboard integration.
Returns a string suitable for enlight grid display."
  (with-temp-buffer
    (prompt-pulse-dashboard-insert)
    (buffer-string)))

(provide 'prompt-pulse)
;;; prompt-pulse.el ends here
`
}
