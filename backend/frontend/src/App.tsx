import React from 'react';
import Core from './modules/core';
import Search from './modules/search';
import Analytics from './modules/analytics';
import Notification from './modules/notification';

export default function App() {
  return (
    <main style={{maxWidth: 960, margin: '40px auto', fontFamily: 'Arial, sans-serif'}}>
      <h1>实现一款提醒事项工具-使用适配移动端的方式开发-提供网页版本 实现骨架</h1>
      <p>该页面由 Super Dev 自动生成，按模块分区承接需求实现。</p>
      <div style={{display: 'grid', gap: 12}}>
        <section style={border: '1px solid #e5e7eb', borderRadius: 12, padding: 16}>
          <h3>Core</h3>
          <Core />
        </section>
        <section style={border: '1px solid #e5e7eb', borderRadius: 12, padding: 16}>
          <h3>Search</h3>
          <Search />
        </section>
        <section style={border: '1px solid #e5e7eb', borderRadius: 12, padding: 16}>
          <h3>Analytics</h3>
          <Analytics />
        </section>
        <section style={border: '1px solid #e5e7eb', borderRadius: 12, padding: 16}>
          <h3>Notification</h3>
          <Notification />
        </section>
      </div>
    </main>
  );
}
