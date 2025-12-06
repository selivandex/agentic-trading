/**
 * Russian translations
 *
 * All user-facing text in Russian language
 */

export const ru = {
  auth: {
    errors: {
      invalidCredentials: "Неверный email или пароль. Проверьте правильность введенных данных.",
      serverError: "Ошибка сервера. Попробуйте позже.",
      authenticationFailed: "Ошибка авторизации. Попробуйте еще раз.",
      genericError: "Не удалось войти в систему. Проверьте правильность данных.",
      emailRequired: "Email обязателен для заполнения.",
      passwordRequired: "Пароль обязателен для заполнения.",
      sessionExpired: "Сессия истекла. Пожалуйста, войдите снова.",
    },
    login: {
      title: "Вход в систему",
      subtitle: "Введите ваши данные для продолжения",
      emailLabel: "Email",
      emailPlaceholder: "Введите ваш email",
      passwordLabel: "Пароль",
      passwordPlaceholder: "Введите ваш пароль",
      submitButton: "Войти",
      submittingButton: "Вход...",
      forgotPassword: "Забыли пароль?",
    },
  },
  common: {
    loading: "Загрузка...",
    error: "Ошибка",
    success: "Успешно",
    cancel: "Отмена",
    save: "Сохранить",
    delete: "Удалить",
    edit: "Редактировать",
    create: "Создать",
    close: "Закрыть",
    yes: "Да",
    no: "Нет",
    ok: "ОК",
  },
  validation: {
    required: "Обязательное поле",
    email: "Некорректный email",
    minLength: "Минимальная длина: {min} символов",
    maxLength: "Максимальная длина: {max} символов",
  },
} as const;

export type Translations = typeof ru;
