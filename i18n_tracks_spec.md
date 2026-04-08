# Specification: Internationalization (i18n) Support in tracks

## Overview
To support multi-language floral studios, the `tracks` framework should provide a robust mechanism for locale detection, translation loading, and context-aware rendering.

## Requirements

### 1. Locale Detection Middleware
The framework should include a middleware that determines the active locale for each request based on the following priority:
1.  **Query Parameter:** `?locale=fr`
2.  **Cookie:** A cookie named `locale` (e.g., `locale=nl`).
3.  **Accept-Language Header:** Parsing the standard browser header.
4.  **Default:** Fallback to `en`.

### 2. Translation Loading
- Automatically load JSON translation files from a `translations/` directory in the project root.
- Expected format: `translations/{locale}.json` (e.g., `en.json`, `fr.json`).
- Support nested JSON structures and access them via dot-notation in the `t` function (e.g., `t "landing.hero.title"`).

### 3. View Context Integration
- Expose the detected locale in the view context so it can be accessed in templates using `v "locale"`.
- This allows layouts to dynamically set `<html lang="{{ v "locale" }}">` and adjust UI elements (like language toggles).

### 4. Template Helper (`t`)
- The `t` function available in templates must automatically use the detected locale from the request context.
- Support for simple interpolation (e.g., `t "greeting" "Name"`) is highly recommended.

### 5. Controller Support
- Provide a way for controllers to easily change the locale (e.g., by setting the `locale` cookie).
- Provide a `T(ctx, key)` function for translated strings within Go code.

## Current Implementation Status
The `floral-crm` project has implemented a `LocaleController` that sets a `locale` cookie. The `tracks` framework needs to ensure it honors this cookie and loads the corresponding translations for subsequent requests.
