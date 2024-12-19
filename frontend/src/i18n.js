import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import en from './locales/en.json';
import zh from './locales/zh.json';

const resources = {
  en: {
    translation: en
  },
  zh: {
    translation: zh
  }
};

i18n
  .use(initReactI18next)
  .init({
    resources,
    lng: localStorage.getItem('language') || 'zh', // 默认语言
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false
    }
  });

export default i18n;
