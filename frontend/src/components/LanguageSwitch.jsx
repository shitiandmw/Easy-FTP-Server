import React from 'react';
import { useTranslation } from 'react-i18next';

export default function LanguageSwitch() {
  const { i18n } = useTranslation();

  const toggleLanguage = () => {
    const newLang = i18n.language === 'zh' ? 'en' : 'zh';
    i18n.changeLanguage(newLang);
    localStorage.setItem('language', newLang);
  };

  return (
    <button
      onClick={toggleLanguage}
      className="px-3 py-1 text-sm text-gray-600 hover:text-gray-800 transition-colors"
    >
      {i18n.language === 'zh' ? 'English' : '中文'}
    </button>
  );
}
