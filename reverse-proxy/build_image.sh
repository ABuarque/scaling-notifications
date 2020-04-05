echo "Building image..."
docker build -t animal505/rp .

echo "Pushing to docker hub..."
docker push animal505/rp
