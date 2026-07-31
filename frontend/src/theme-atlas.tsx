import React from 'react'
import {createRoot} from 'react-dom/client'

import ThemeAtlas from './ThemeAtlas'

const container = document.getElementById('root')

createRoot(container!).render(
  <React.StrictMode>
    <ThemeAtlas/>
  </React.StrictMode>,
)
