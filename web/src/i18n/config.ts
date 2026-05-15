import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import ko from './ko.json';
import en from './en.json';

i18n.use(initReactI18next).init({
  resources: {
    ko: { translation: ko },
    en: { translation: en },
  },
  lng: 'ko',           // 한국어 기본
  fallbackLng: 'ko',
  interpolation: {
    escapeValue: false, // React는 XSS 방어를 자체 처리
  },
});

export default i18n;
