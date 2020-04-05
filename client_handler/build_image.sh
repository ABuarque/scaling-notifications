echo "Building image..."
docker build -t animal505/sse .

echo "Pushing to docker hub..."
docker push animal505/sse
