"use client";

import React, { createContext, useState, useContext } from 'react';

interface Session {
  username: string;
  isAuthenticated: boolean;
}

interface SessionContextType {
  session: Session;
  login: (username: string) => void;
  logout: () => void;
}

const SessionContext = createContext<SessionContextType | undefined>(undefined);

export const SessionProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [session, setSession] = useState<Session>({ username: '', isAuthenticated: false });

  const login = (username: string) => {
    setSession({ username, isAuthenticated: true });
  };

  const logout = () => {
    setSession({ username: '', isAuthenticated: false });
  };

  return (
    <SessionContext.Provider value={{ session, login, logout }}>
      {children}
    </SessionContext.Provider>
  );
};

export const useSession = () => {
  const context = useContext(SessionContext);
  if (!context) {
    throw new Error('useSession debe usarse dentro de SessionProvider');
  }
  return context;
};