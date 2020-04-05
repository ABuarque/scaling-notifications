echo "Building image..."
docker build -t animal505/dispatcher .

echo "Pushing to docker hub..."
docker push animal505/dispatcher
