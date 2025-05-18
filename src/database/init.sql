-- Initialize CinemaAbyss Database Schema

-- Create database if it doesn't exist
-- Note: This is handled by Docker Compose and Kubernetes configs

-- Connect to the database
\c cinemaabyss;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create movies table
CREATE TABLE IF NOT EXISTS movies (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    release_year INTEGER,
    duration INTEGER,
    rating DECIMAL(3,1),
    genres TEXT[],
    director VARCHAR(255),
    actors TEXT[],
    poster_url VARCHAR(255),
    trailer_url VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create movie_genres table for many-to-many relationship
CREATE TABLE IF NOT EXISTS movie_genres (
    movie_id INTEGER REFERENCES movies(id) ON DELETE CASCADE,
    genre VARCHAR(50) NOT NULL,
    PRIMARY KEY (movie_id, genre)
);

-- Create payments table
CREATE TABLE IF NOT EXISTS payments (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(10, 2) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(50) DEFAULT 'completed',
    payment_method VARCHAR(50),
    transaction_id VARCHAR(255)
);

-- Create subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    plan_type VARCHAR(50) NOT NULL,
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE NOT NULL,
    auto_renew BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create views table to track movie views
CREATE TABLE IF NOT EXISTS views (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    movie_id INTEGER REFERENCES movies(id) ON DELETE CASCADE,
    view_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    duration INTEGER, -- how many seconds the user watched
    completed BOOLEAN DEFAULT FALSE
);

-- Create user_ratings table
CREATE TABLE IF NOT EXISTS user_ratings (
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    movie_id INTEGER REFERENCES movies(id) ON DELETE CASCADE,
    rating INTEGER CHECK (rating >= 1 AND rating <= 10),
    review TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, movie_id)
);

-- Insert some sample data

-- Sample users
INSERT INTO users (username, email) VALUES
    ('user1', 'user1@example.com'),
    ('user2', 'user2@example.com'),
    ('user3', 'user3@example.com')
ON CONFLICT (username) DO NOTHING;

-- Sample movies
INSERT INTO movies (title, description, release_year, duration, rating, genres, director, actors, poster_url, trailer_url)
VALUES 
    ('Inception', 'A thief who steals corporate secrets through dream-sharing technology is given the inverse task of planting an idea into the mind of a C.E.O.', 2010, 148, 8.8, ARRAY['Sci-Fi', 'Action', 'Thriller'], 'Christopher Nolan', ARRAY['Leonardo DiCaprio', 'Joseph Gordon-Levitt', 'Ellen Page'], 'https://example.com/inception.jpg', 'https://example.com/inception-trailer.mp4'),
    ('The Shawshank Redemption', 'Two imprisoned men bond over a number of years, finding solace and eventual redemption through acts of common decency.', 1994, 142, 9.3, ARRAY['Drama'], 'Frank Darabont', ARRAY['Tim Robbins', 'Morgan Freeman'], 'https://example.com/shawshank.jpg', 'https://example.com/shawshank-trailer.mp4'),
    ('The Dark Knight', 'When the menace known as the Joker wreaks havoc and chaos on the people of Gotham, Batman must accept one of the greatest psychological and physical tests of his ability to fight injustice.', 2008, 152, 9.0, ARRAY['Action', 'Crime', 'Drama'], 'Christopher Nolan', ARRAY['Christian Bale', 'Heath Ledger', 'Aaron Eckhart'], 'https://example.com/dark-knight.jpg', 'https://example.com/dark-knight-trailer.mp4'),
    ('Pulp Fiction', 'The lives of two mob hitmen, a boxer, a gangster and his wife, and a pair of diner bandits intertwine in four tales of violence and redemption.', 1994, 154, 8.9, ARRAY['Crime', 'Drama'], 'Quentin Tarantino', ARRAY['John Travolta', 'Samuel L. Jackson', 'Uma Thurman'], 'https://example.com/pulp-fiction.jpg', 'https://example.com/pulp-fiction-trailer.mp4'),
    ('Forrest Gump', 'The presidencies of Kennedy and Johnson, the Vietnam War, the Watergate scandal and other historical events unfold from the perspective of an Alabama man with an IQ of 75, whose only desire is to be reunited with his childhood sweetheart.', 1994, 142, 8.8, ARRAY['Drama', 'Romance'], 'Robert Zemeckis', ARRAY['Tom Hanks', 'Robin Wright', 'Gary Sinise'], 'https://example.com/forrest-gump.jpg', 'https://example.com/forrest-gump-trailer.mp4')
ON CONFLICT (id) DO NOTHING;

-- Sample movie genres
INSERT INTO movie_genres (movie_id, genre) VALUES
    (1, 'Drama'),
    (2, 'Crime'),
    (2, 'Drama'),
    (3, 'Action'),
    (3, 'Crime'),
    (3, 'Drama'),
    (4, 'Crime'),
    (4, 'Drama'),
    (5, 'Drama'),
    (5, 'Romance')
ON CONFLICT (movie_id, genre) DO NOTHING;

-- Sample payments
INSERT INTO payments (user_id, amount) VALUES
    (1, 9.99),
    (2, 14.99),
    (3, 19.99)
ON CONFLICT (id) DO NOTHING;

-- Sample subscriptions
INSERT INTO subscriptions (user_id, plan_type, start_date, end_date) VALUES
    (1, 'basic', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP + INTERVAL '30 days'),
    (2, 'premium', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP + INTERVAL '30 days'),
    (3, 'family', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP + INTERVAL '30 days')
ON CONFLICT (id) DO NOTHING;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_movies_title ON movies(title);
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_views_user_id ON views(user_id);
CREATE INDEX IF NOT EXISTS idx_views_movie_id ON views(movie_id);
CREATE INDEX IF NOT EXISTS idx_movies_release_year ON movies(release_year);
CREATE INDEX IF NOT EXISTS idx_movies_rating ON movies(rating);