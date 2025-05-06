import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import IndexRouter from './routes/IndexRouter';
import AlbumRouter from './routes/AlbumRouter';

function App() {
    return (
        <Router>
            <Routes>
                <Route path='/album/:identifier/*' element={<AlbumRouter />} />
                <Route path='/*' element={<IndexRouter />} />
            </Routes>
        </Router>
    );
}

export default App;
